package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	"github.com/qmish/FProto/server/internal/crypto"
	"github.com/qmish/FProto/server/internal/observability"
	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
	"github.com/qmish/FProto/server/internal/transport"
)

var (
	logger  *zap.Logger
	metrics *observability.Metrics
)

func main() {
	wsAddr := flag.String("ws-addr", ":8080", "WebSocket listen address")
	quicAddr := flag.String("quic-addr", ":8443", "QUIC listen address")
	metricsAddr := flag.String("metrics", ":9090", "Prometheus metrics address")
	dispatcherAddr := flag.String("dispatcher", "localhost:50053", "Dispatcher gRPC address")
	otlpEndpoint := flag.String("otlp", "localhost:4317", "OTLP gRPC endpoint")
	flag.Parse()

	var err error
	logger, err = observability.NewLogger("gateway")
	if err != nil {
		panic("logger init: " + err.Error())
	}
	defer func() { _ = logger.Sync() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownTracer, err := observability.InitTracer(ctx, "gateway", "0.5.0", *otlpEndpoint)
	if err != nil {
		logger.Warn("Tracing init failed, continuing without traces", zap.Error(err))
	} else {
		defer func() { _ = shutdownTracer(ctx) }()
	}

	mp, metricsHandler, err := observability.InitMeter()
	if err != nil {
		logger.Fatal("Metrics init failed", zap.Error(err))
	}
	defer func() { _ = mp.Shutdown(ctx) }()

	metrics, err = observability.NewMetrics("gateway")
	if err != nil {
		logger.Fatal("App metrics init failed", zap.Error(err))
	}

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", metricsHandler)
		mux.HandleFunc("/debug/pprof/", http.DefaultServeMux.ServeHTTP)
		logger.Info("Metrics + pprof endpoint", zap.String("addr", *metricsAddr))
		if err := http.ListenAndServe(*metricsAddr, mux); err != nil {
			logger.Error("Metrics server error", zap.Error(err))
		}
	}()

	serverKey, err := crypto.GenerateKeyPair()
	if err != nil {
		logger.Fatal("Ошибка генерации ключей", zap.Error(err))
	}
	logger.Info("Gateway публичный ключ сгенерирован")

	var dispatcherClient pb.DispatcherServiceClient
	if *dispatcherAddr != "" {
		dialOpts := append(observability.GRPCDialOptions(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		conn, err := grpc.NewClient(*dispatcherAddr, dialOpts...)
		if err != nil {
			logger.Warn("Dispatcher gRPC connect failed, echo mode", zap.Error(err))
		} else {
			dispatcherClient = pb.NewDispatcherServiceClient(conn)
			logger.Info("Gateway -> Dispatcher", zap.String("addr", *dispatcherAddr))
		}
	}

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws, err := transport.UpgradeHTTP(w, r)
		if err != nil {
			logger.Warn("WS upgrade error", zap.Error(err))
			return
		}
		defer ws.Close()
		handleConn(ctx, ws, serverKey, dispatcherClient)
	})

	go func() {
		logger.Info("WebSocket Gateway", zap.String("addr", *wsAddr))
		if err := http.ListenAndServe(*wsAddr, nil); err != nil {
			logger.Error("WS server error", zap.Error(err))
		}
	}()

	tlsConf := transport.GenerateSelfSignedTLS()
	go func() {
		listener, err := transport.ListenQUIC(*quicAddr, tlsConf)
		if err != nil {
			logger.Error("QUIC listen error", zap.Error(err))
			return
		}
		logger.Info("QUIC Gateway", zap.String("addr", *quicAddr))
		for {
			qconn, err := transport.AcceptQUIC(ctx, listener)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				logger.Warn("QUIC accept error", zap.Error(err))
				continue
			}
			go func() {
				defer qconn.Close()
				handleConn(ctx, qconn, serverKey, dispatcherClient)
			}()
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	logger.Info("Gateway завершает работу...")
	cancel()
}

func handleConn(ctx context.Context, conn transport.Conn, serverKey *crypto.KeyPair, dispatcher pb.DispatcherServiceClient) {
	tracer := observability.Tracer("gateway")
	transportType := string(conn.Type())
	transportAttr := attribute.String("transport", transportType)

	connCtx, connSpan := tracer.Start(ctx, "gateway.connection",
		trace.WithAttributes(transportAttr, attribute.String("remote_addr", conn.RemoteAddr())),
	)
	defer connSpan.End()

	metrics.ActiveConnections.Add(connCtx, 1, otelmetric.WithAttributes(transportAttr))
	defer metrics.ActiveConnections.Add(connCtx, -1, otelmetric.WithAttributes(transportAttr))

	hsStart := time.Now()
	_, hsSpan := tracer.Start(connCtx, "gateway.handshake")
	session, err := crypto.ServerHandshake(serverKey, conn.ReadMessage, conn.WriteMessage)
	hsDuration := time.Since(hsStart).Seconds()
	metrics.HandshakeDuration.Record(connCtx, hsDuration, otelmetric.WithAttributes(transportAttr))

	if err != nil {
		hsSpan.RecordError(err)
		hsSpan.SetStatus(codes.Error, err.Error())
		hsSpan.End()
		logger.Warn("Handshake error",
			zap.String("transport", transportType),
			zap.Error(err))
		return
	}
	hsSpan.End()
	logger.Info("Handshake OK",
		zap.String("transport", transportType),
		zap.String("remote_addr", conn.RemoteAddr()))

	for {
		ct, err := conn.ReadMessage()
		if err != nil {
			return
		}
		metrics.MessagesReceived.Add(connCtx, 1, otelmetric.WithAttributes(transportAttr))

		_, decSpan := tracer.Start(connCtx, "gateway.decrypt")
		pt, err := session.Decrypt(ct)
		if err != nil {
			decSpan.RecordError(err)
			decSpan.SetStatus(codes.Error, err.Error())
			decSpan.End()
			logger.Warn("Decrypt error",
				zap.String("transport", transportType),
				zap.Error(err))
			return
		}
		decSpan.End()

		if dispatcher != nil {
			var frame pb.Frame
			if err := proto.Unmarshal(pt, &frame); err == nil && len(frame.EncryptedPayload) > 0 {
				var appMsg pb.AppMessage
				if err := proto.Unmarshal(frame.EncryptedPayload, &appMsg); err == nil {
					_, dispSpan := tracer.Start(connCtx, "gateway.dispatch")
					resp, err := dispatcher.Dispatch(connCtx, &pb.DispatchRequest{
						SessionId: []byte(conn.RemoteAddr()),
						SenderId:  frame.SessionId,
						Message:   &appMsg,
					})
					if err != nil {
						dispSpan.RecordError(err)
						dispSpan.SetStatus(codes.Error, err.Error())
						dispSpan.End()
						logger.Warn("Dispatch error",
							zap.String("transport", transportType),
							zap.Error(err))
					} else {
						dispSpan.End()
						metrics.MessagesSent.Add(connCtx, 1, otelmetric.WithAttributes(transportAttr))
						ack := &pb.AppMessage{
							MessageId: resp.AckMessageId,
							Body: &pb.AppMessage_Ack{
								Ack: &pb.Ack{
									AckMessageId: resp.AckMessageId,
									Status:       pb.Ack_RECEIVED,
								},
							},
						}
						ackBytes, _ := proto.Marshal(ack)
						enc, err := session.Encrypt(ackBytes)
						if err == nil {
							_ = conn.WriteMessage(enc)
						}
						continue
					}
				}
			}
		}

		resp := fmt.Appendf(nil, "echo: %s", pt)
		enc, err := session.Encrypt(resp)
		if err != nil {
			return
		}
		if err := conn.WriteMessage(enc); err != nil {
			return
		}
		metrics.MessagesSent.Add(connCtx, 1, otelmetric.WithAttributes(transportAttr))
	}
}

func init() {
	_ = tls.Config{}
}

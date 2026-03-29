plugins {
    kotlin("jvm") version "1.9.22"
}

group = "com.fproto"
version = "0.2.0"

repositories {
    mavenCentral()
}

dependencies {
    implementation("com.squareup.okhttp3:okhttp:4.12.0")
    implementation("org.whispersystems:curve25519-java:0.5.0")

    testImplementation(kotlin("test"))
    testImplementation("org.junit.jupiter:junit-jupiter:5.10.2")
    testImplementation("org.java-websocket:Java-WebSocket:1.5.6")
}

tasks.test {
    useJUnitPlatform()
}

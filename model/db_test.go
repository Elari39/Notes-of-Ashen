package model

import (
	"errors"
	"net"
	"strings"
	"testing"
)

func TestMysqlConnectErrorIncludesSafeRemoteTarget(t *testing.T) {
	cause := errors.New("unexpected EOF")
	err := mysqlConnectError("notes_user:super-secret@tcp(mysql.example.com:3306)/notes_of_ashen?charset=utf8mb4&parseTime=true&loc=Local", cause)

	if !errors.Is(err, cause) {
		t.Fatal("expected mysqlConnectError to wrap the original error")
	}

	msg := err.Error()
	for _, want := range []string{
		"mysql.example.com:3306",
		"notes_of_ashen",
		"unexpected EOF",
		"APP_DATABASE_DSN",
		"charset=utf8mb4&parseTime=true&loc=Local",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error message %q does not contain %q", msg, want)
		}
	}

	for _, leaked := range []string{"super-secret", "notes_user"} {
		if strings.Contains(msg, leaked) {
			t.Fatalf("error message leaks DSN credential fragment %q: %q", leaked, msg)
		}
	}
}

func TestMysqlConnectErrorDoesNotLeakMalformedDSN(t *testing.T) {
	rawDSN := "notes_user:super-secret@tcp(mysql.example.com:3306)/notes_of_ashen?loc=%ZZ"
	err := mysqlConnectError(rawDSN, errors.New("invalid connection"))

	msg := err.Error()
	if !strings.Contains(msg, "invalid APP_DATABASE_DSN format") {
		t.Fatalf("error message %q does not mention invalid APP_DATABASE_DSN format", msg)
	}

	for _, leaked := range []string{rawDSN, "super-secret", "notes_user"} {
		if strings.Contains(msg, leaked) {
			t.Fatalf("error message leaks malformed DSN fragment %q: %q", leaked, msg)
		}
	}
}

func TestMysqlConnectErrorIncludesProbeDiagnosis(t *testing.T) {
	cause := errors.New("invalid connection")
	probeCalled := false
	err := mysqlConnectErrorWithProbe(
		"notes_user:super-secret@tcp(mysql.example.com:3306)/notes_of_ashen?charset=utf8mb4&parseTime=true&loc=Local",
		cause,
		func(target mysqlTarget) string {
			probeCalled = true
			if target.network != "tcp" {
				t.Fatalf("target.network = %q", target.network)
			}
			if target.address != "mysql.example.com:3306" {
				t.Fatalf("target.address = %q", target.address)
			}
			return "diagnosis: TCP connection is accepted, but the server closes before sending a MySQL handshake"
		},
	)

	if !probeCalled {
		t.Fatal("expected mysqlConnectErrorWithProbe to call probe")
	}
	if !errors.Is(err, cause) {
		t.Fatal("expected mysqlConnectErrorWithProbe to wrap the original error")
	}

	msg := err.Error()
	if !strings.Contains(msg, "server closes before sending a MySQL handshake") {
		t.Fatalf("error message %q does not contain probe diagnosis", msg)
	}
	for _, leaked := range []string{"super-secret", "notes_user"} {
		if strings.Contains(msg, leaked) {
			t.Fatalf("error message leaks DSN credential fragment %q: %q", leaked, msg)
		}
	}
}

func TestProbeMySQLHandshakeDetectsCloseBeforeHandshake(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	accepted := make(chan struct{})
	go func() {
		defer close(accepted)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		_ = conn.Close()
	}()

	diagnosis := probeMySQLHandshake(mysqlTarget{
		network:  "tcp",
		address:  listener.Addr().String(),
		endpoint: listener.Addr().String(),
		database: "notes_of_ashen",
	})

	<-accepted
	if !strings.Contains(diagnosis, "server closes before sending a MySQL handshake") {
		t.Fatalf("diagnosis %q does not describe closed connection before handshake", diagnosis)
	}
}

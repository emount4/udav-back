package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/emount4/udav-back/internal/core/db"
	"github.com/jackc/pgx/v5/pgproto3"
)

const (
	testDSN       = "postgres://udav@127.0.0.1/udav?sslmode=disable"
	testUpdate    = "update profiles set version = version + 1"
	testSelect    = "select 1"
	testEmptyRows = "select 1 where false"
)

func TestLoadConfigValidatesPoolLimitsWithoutExposingDSN(t *testing.T) {
	tests := []struct {
		name, dsn, message string
	}{
		{"invalid DSN", "postgres://udav:secret%zz@localhost/udav", "invalid DATABASE_URL"},
		{"inverted limits", testDSN + "&pool_max_conns=2&pool_min_conns=3", "pool_min_conns"},
		{"invalid health check", testDSN + "&pool_health_check_period=0s", "pool_health_check_period"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				message := fmt.Sprint(recover())
				if !strings.Contains(message, test.message) || strings.Contains(message, "secret") {
					t.Fatalf("unexpected config error: %s", message)
				}
			}()
			MustLoadConfig(func(string) string { return test.dsn })
		})
	}
}

func TestOpenExposesDriverIndependentQueries(t *testing.T) {
	field := pgproto3.FieldDescription{Name: []byte("value"), DataTypeOID: 23, DataTypeSize: 4}
	oneRow := testQuery{testSelect, 'I', []pgproto3.BackendMessage{
		&pgproto3.RowDescription{Fields: []pgproto3.FieldDescription{field}},
		&pgproto3.DataRow{Values: [][]byte{[]byte("1")}},
		&pgproto3.CommandComplete{CommandTag: []byte("SELECT 1")},
	}}
	pool := openTestPool(t,
		command(testUpdate, "UPDATE 2", 'I'),
		oneRow,
		testQuery{testEmptyRows, 'I', []pgproto3.BackendMessage{
			&pgproto3.RowDescription{Fields: []pgproto3.FieldDescription{field}},
			&pgproto3.CommandComplete{CommandTag: []byte("SELECT 0")},
		}},
		oneRow,
	)

	var executor db.DBTX = pool
	result, err := executor.Exec(t.Context(), testUpdate)
	if err != nil || result == nil || result.RowsAffected() != 2 {
		t.Fatalf("unexpected update result: %v, %v", result, err)
	}
	rows, err := executor.Query(t.Context(), testSelect)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var value int
	if !rows.Next() {
		t.Fatalf("expected a row: %v", rows.Err())
	}
	if err := rows.Scan(&value); err != nil || value != 1 {
		t.Fatalf("unexpected row: %d, %v", value, err)
	}
	if rows.Next() || rows.Err() != nil {
		t.Fatalf("expected the end of the result: %v", rows.Err())
	}
	if err := executor.QueryRow(t.Context(), testEmptyRows).Scan(&value); !errors.Is(err, sql.ErrNoRows) || !strings.HasPrefix(err.Error(), "scan postgres row:") {
		t.Fatalf("expected a scan error wrapping sql.ErrNoRows, got %v", err)
	}

	badRows, err := executor.Query(t.Context(), testSelect)
	if err != nil {
		t.Fatal(err)
	}
	defer badRows.Close()
	if !badRows.Next() {
		t.Fatalf("expected a row: %v", badRows.Err())
	}
	scanErr := badRows.Scan()
	cause := errors.Unwrap(scanErr)
	if cause == nil || !strings.HasPrefix(scanErr.Error(), "scan postgres rows:") {
		t.Fatalf("expected a wrapped scan error, got %v", scanErr)
	}
	if err := badRows.Err(); !errors.Is(err, cause) || !strings.HasPrefix(err.Error(), "iterate postgres rows:") {
		t.Fatalf("expected an iteration error wrapping %v, got %v", cause, err)
	}

	if err := pool.Close(); err != nil {
		t.Fatal(err)
	}
	_, execErr := executor.Exec(t.Context(), testUpdate)
	_, queryErr := executor.Query(t.Context(), testSelect)
	for _, failure := range []struct {
		operation string
		err       error
	}{
		{"execute postgres query:", execErr},
		{"query postgres rows:", queryErr},
		{"ping postgres:", pool.Ping(t.Context())},
	} {
		if errors.Unwrap(failure.err) == nil || !strings.HasPrefix(failure.err.Error(), failure.operation) {
			t.Fatalf("expected a wrapped %q error, got %v", failure.operation, failure.err)
		}
	}
}

func TestOpenReturnsConnectionError(t *testing.T) {
	cfg := MustLoadConfig(func(string) string { return testDSN })
	dialErr := errors.New("unreachable")
	cfg.pool.ConnConfig.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		return nil, dialErr
	}

	pool, err := Open(t.Context(), cfg)
	if pool != nil || !errors.Is(err, dialErr) || !strings.HasPrefix(err.Error(), "ping postgres:") {
		t.Fatalf("expected a connection error, got %v", err)
	}
}

type testQuery struct {
	sql      string
	txStatus byte
	response []pgproto3.BackendMessage
}

func command(query, tag string, txStatus byte) testQuery {
	return testQuery{query, txStatus, []pgproto3.BackendMessage{&pgproto3.CommandComplete{CommandTag: []byte(tag)}}}
}

func openTestPool(t *testing.T, queries ...testQuery) db.Pool {
	t.Helper()
	cfg := MustLoadConfig(func(string) string {
		return testDSN + "&pool_max_conns=1&default_query_exec_mode=simple_protocol"
	})
	ping := testQuery{"-- ping", 'I', []pgproto3.BackendMessage{&pgproto3.EmptyQueryResponse{}}}
	script := append([]testQuery{ping}, queries...)
	peerResult := make(chan error, 1)
	cfg.pool.ConnConfig.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		client, peer := net.Pipe()
		go func() {
			defer peer.Close()
			peerResult <- servePoolQueries(peer, script)
		}()
		return client, nil
	}

	pool, err := Open(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pool.Close(); err != nil {
			t.Error(err)
		}
		if err := <-peerResult; err != nil {
			t.Error(err)
		}
	})
	return pool
}

func servePoolQueries(peer net.Conn, queries []testQuery) error {
	backend := pgproto3.NewBackend(peer, peer)
	if _, err := backend.ReceiveStartupMessage(); err != nil {
		return err
	}
	backend.Send(&pgproto3.AuthenticationOk{})
	backend.Send(&pgproto3.ParameterStatus{Name: "standard_conforming_strings", Value: "on"})
	backend.Send(&pgproto3.ParameterStatus{Name: "client_encoding", Value: "UTF8"})
	backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
	if err := backend.Flush(); err != nil {
		return err
	}

	for _, query := range queries {
		message, err := backend.Receive()
		if err != nil {
			return err
		}
		if received, ok := message.(*pgproto3.Query); !ok || received.String != query.sql {
			return fmt.Errorf("expected query %q, got %v", query.sql, message)
		}
		for _, message := range query.response {
			backend.Send(message)
		}
		backend.Send(&pgproto3.ReadyForQuery{TxStatus: query.txStatus})
		if err := backend.Flush(); err != nil {
			return err
		}
	}

	message, err := backend.Receive()
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, ok := message.(*pgproto3.Terminate); !ok {
		return fmt.Errorf("expected connection termination, got %T", message)
	}
	return nil
}

package backtest

import (
    "bytes"
    "context"
    "testing"
)

func TestCSVKlineFeeder(t *testing.T) {
    data := []byte("ts,close\n1,100\n2,101\n3,102\n")
    feeder, err := NewCSVKlineFeeder("BTC", bytes.NewReader(data))
    if err != nil { t.Fatalf("NewCSVKlineFeeder: %v", err) }
    ctx := context.Background()

    snap, ok, err := feeder.Next(ctx, "BTC")
    if err != nil || !ok { t.Fatalf("Next1: %v ok=%v", err, ok) }
    if snap.Price.Last != 100 { t.Fatalf("px1=%v", snap.Price.Last) }

    snap, ok, err = feeder.Next(ctx, "BTC")
    if err != nil || !ok { t.Fatalf("Next2: %v ok=%v", err, ok) }
    if snap.Price.Last != 101 { t.Fatalf("px2=%v", snap.Price.Last) }
    if snap.Change.OneHour <= 0 { t.Fatalf("expected positive change") }

    _, ok, err = feeder.Next(ctx, "BTC")
    if err != nil || !ok { t.Fatalf("Next3: %v ok=%v", err, ok) }

    _, ok, err = feeder.Next(ctx, "BTC")
    if err != nil || ok { t.Fatalf("expected eof: ok=%v err=%v", ok, err) }
}


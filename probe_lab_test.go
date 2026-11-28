package funcstore

import "testing"

func TestProbeDateFormats(t *testing.T) {
	r := NewPhpscriptRuntime()
	prog, err := r.Compile(`<?php
$ts = strtotime('2016-02-15');
echo date('Y', $ts) . '|' . date('m', $ts) . '|' . date('d', $ts) . '|' . date('n', $ts) . '|' . date('w', $ts) . '|' . date('t', $ts) . '|' . date('Y-m-d', $ts) . '|' . date('H:i:s', $ts);
`)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	out, err := prog.Run(map[string]any{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	m := out.(map[string]interface{})
	t.Logf("body=%q", m["body"])
}

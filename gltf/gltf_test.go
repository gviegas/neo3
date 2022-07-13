// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package gltf

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestGLTF(t *testing.T) {
	file, err := os.Open("testdata/cube.gltf")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gltf, err := Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	err = Encode(&buf, gltf)
	if err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	buf.Reset()
	err = json.Indent(&buf, []byte(s), "", "    ")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(buf.Bytes()))
}

func TestGLB(t *testing.T) {
	file, err := os.Open("testdata/cube.glb")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if !IsGLB(file) {
		t.Fatal("IsGLB(file):\nwant true\nhave false")
	}
	r := bytes.NewReader([]byte(`{"asset:"{"version":"2.0"}}`))
	if IsGLB(r) {
		t.Fatal("IsGLB(r):\nwant false\nhave true")
	}
	file.Seek(0, 0)
	if !IsGLB(file) {
		t.Fatal("IsGLB(file):\nwant true\nhave false")
	}
}

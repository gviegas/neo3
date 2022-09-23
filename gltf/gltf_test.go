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

	// SeekJSON
	//file.Seek(0, 0)
	n, err := SeekJSON(file, 1)
	if err != nil {
		t.Fatal(err)
	}
	if n <= 0 {
		t.Fatal("SeekJSON(file):\nwant n > 0\nhave 0")
	}
	b := make([]byte, n)
	n, err = file.Read(b)
	if err != nil {
		t.Fatal(err)
	}
	gltf, err := Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	err = Encode(&buf, gltf)
	if err != nil {
		t.Fatal(err)
	}
	b = append(b[:0], buf.Bytes()...)
	buf.Reset()
	err = json.Indent(&buf, b, "", "    ")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(buf.Bytes()))

	// SeekBIN
	//file.Seek(0, 0)
	n, err = SeekBIN(file, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := gltf.Buffers[0].ByteLength
	if pad := want % 4; pad != 0 {
		want += 4 - pad
	}
	if want != int64(n) {
		t.Fatalf("SeekBIN(file):\nwant n == %d\nhave n == %d", want, n)
	}
	if n > len(b) {
		b = make([]byte, n)
	} else {
		b = b[:n]
	}
	n, err = file.Read(b)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPack(t *testing.T) {
	file, err := os.Open("testdata/cube.gltf")
	if err != nil {
		t.Fatal(err)
	}
	gltf, err := Decode(file)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	gltf.Asset.Generator = "TestPack"
	var buf bytes.Buffer
	file, err = os.Open("testdata/cube.bin")
	if err != nil {
		t.Fatal(err)
	}
	n, err := buf.ReadFrom(file)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	gltf.Buffers[0] = Buffer{ByteLength: n}
	file, err = os.Create("testdata/out.glb")
	if err != nil {
		t.Fatal(err)
	}
	err = Pack(file, gltf, buf.Bytes())
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
}

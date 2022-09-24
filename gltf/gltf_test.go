// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package gltf

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"
)

// This needs to match binary buffer length from both
// testdata/cube.glb and testdata/cube.bin, ignoring
// padding bytes.
// Also, the contents of testdata/cube.bin must be
// identical to those in testdata/cube.glb's BIN chunk.
const cubeByteLen = 840

func TestMinimalGLTF(t *testing.T) {
	r := bytes.NewReader([]byte(`{"asset":{"version":"2.0"}}`))
	gltf, err := Decode(r)
	if err != nil {
		t.Fatal(err)
	}
	if s := gltf.Asset.Version; s != "2.0" {
		t.Fatalf("Decode(r): gltf.Asset.Version:\nwant 2.0\nhave %s", s)
	}
	var buf bytes.Buffer
	if err = Encode(&buf, gltf); err != nil {
		t.Fatal(err)
	}
	r.Seek(0, 0)
	n := int(r.Size())
	if buf.Len()-1 == n {
		s := buf.String()
		for ; n > 0; n-- {
			b1, err1 := r.ReadByte()
			b2, err2 := buf.ReadByte()
			if b1 != b2 {
				t.Fatal("Encode(&buf, gltf):\ncontent mismatch")
			}
			if err1 != nil || err2 != nil {
				if n == 1 && err1 == io.EOF {
					break
				} else {
					t.Fatal(err1, err2)
				}
			}
		}
		t.Log(s)
		return
	}
	t.Fatalf("Encode(&buf, gltf): buf.Len()\nwant %d\nhave %d", n+1, buf.Len())
}

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

func TestIsGLB(t *testing.T) {
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
}

func TestSeekJSON(t *testing.T) {
	file, err := os.Open("testdata/cube.glb")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	// From the beginning of the GLB.
	n, err := SeekJSON(file, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}
	if n <= 0 {
		t.Fatalf("SeekJSON(file): n:\nwant > 0\nhave %d", n)
	}
	b := make([]byte, n)
	if x, err := file.Read(b); err != nil {
		if x != n || err != io.EOF {
			t.Fatal(err)
		}
	}
	gltf, err := Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err = Encode(&buf, gltf); err != nil {
		t.Fatal(err)
	}
	nprev := n
	sprev := buf.String()
	buf.Reset()
	// From the beginning of the JSON chunk.
	file.Seek(0, 0)
	IsGLB(file)
	n, err = SeekJSON(file, io.SeekCurrent)
	if err != nil {
		t.Fatal(err)
	}
	if n != nprev {
		t.Fatalf("SeekJSON(file): n:\nwant %d\nhave %d", nprev, n)
	}
	if x, err := file.Read(b); err != nil {
		if x != n || err != io.EOF {
			t.Fatal(err)
		}
	}
	if gltf, err = Decode(bytes.NewReader(b)); err != nil {
		t.Fatal(err)
	}
	if err = Encode(&buf, gltf); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if s != sprev {
		t.Fatalf("SeekJson(file): Decode/Encode:\nwant %s\nhave %s", sprev, s)
	}
}

func TestSeekBIN(t *testing.T) {
	file, err := os.Open("testdata/cube.glb")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	n, err := SeekJSON(file, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}
	if n <= 0 {
		t.Fatalf("SeekJSON(file): n:\nwant > 0\nhave %d", n)
	}
	b := make([]byte, n)
	if x, err := file.Read(b); err != nil {
		t.Fatal(err)
	} else if x != n {
		t.Fatal()
	}
	gltf, err := Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	nwant := gltf.Buffers[0].ByteLength
	if pad := nwant % 4; pad != 0 {
		nwant += 4 - pad
	}
	// From the beginning of the BIN chunk.
	n, err = SeekBIN(file, io.SeekCurrent)
	if err != nil {
		t.Fatal(err)
	}
	if nwant != int64(n) {
		t.Fatalf("SeekBIN(file): n:\nwant %d\nhave %d", nwant, n)
	}
	if n > len(b) {
		b = make([]byte, n)
	} else {
		b = b[:n]
	}
	if x, err := file.Read(b); err != nil {
		if x != n || err != io.EOF {
			t.Fatal(err)
		}
	}
	// From the beginning of the GLB.
	file.Seek(0, 0)
	n, err = SeekBIN(file, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}
	if nwant != int64(n) {
		t.Fatalf("SeekBIN(file): n:\nwant %d\nhave %d", nwant, n)
	}
	if x, err := file.Read(b); err != nil {
		if x != n || err != io.EOF {
			t.Fatal(err)
		}
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

func TestUnpack(t *testing.T) {
	file, err := os.Open("testdata/cube.glb")
	if err != nil {
		t.Fatal(err)
	}
	gltf, bin, err := Unpack(file)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err = Encode(&buf, gltf); err != nil {
		t.Fatal(err)
	}
	if gltf.Buffers[0].ByteLength != cubeByteLen {
		panic("gltf tests must be kept in sync with testdata/cube.*")
	}
	file, err = os.Open("testdata/cube.bin")
	if err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	n, err := buf.ReadFrom(file)
	file.Close()
	if n < gltf.Buffers[0].ByteLength && err != nil {
		t.Fatal(err)
	}
	b1 := (*[cubeByteLen]byte)(bin[:cubeByteLen])
	b2 := (*[cubeByteLen]byte)(buf.Bytes()[:cubeByteLen])
	if *b1 != *b2 {
		t.Fatal("Unpack(file):\nbinary buffer mismatch")
	}
}

func TestNoBINChunk(t *testing.T) {
	var gltf GLTF
	gltf.Asset.Generator = "TestNoBINChunk"
	gltf.Asset.Version = "2.0"
	gltf.Nodes = append(gltf.Nodes, Node{Name: "Node#0"})
	var buf bytes.Buffer
	if err := Encode(&buf, &gltf); err != nil {
		t.Fatal()
	}
	s := buf.String()
	buf.Reset()
	if err := Pack(&buf, &gltf, nil); err != nil {
		t.Fatal()
	}
	tf, bin, err := Unpack(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(bin); n != 0 {
		t.Fatalf("Unpack(&buf): len(bin):\nwant 0\nhave %d", n)
	}
	if err = Encode(&buf, tf); err != nil {
		t.Fatal(err)
	}
	if x := buf.String(); x != s {
		t.Fatalf("Unpack(&buf): Encode(&buf, tf):\nwant %s\nhave %s", s, x)
	}
}

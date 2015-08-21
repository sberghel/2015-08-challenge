package main

import (
	"bufio"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type HexReader struct {
	File      io.Reader
	buffer    []byte
	bytesRead int
}

func NewHexReader(f io.Reader) HexReader {
	writer := HexReader{
		File: f,
	}
	return writer
}

func (h *HexReader) Read(p []byte) (int, error) {
	if h == nil || h.File == nil {
		return 0, errors.New("cannot write to nil HexReader")
	}

	var bytesInP int
	var leftover []byte
	var err error
	for bytesInP < len(p) && err == nil {
		var n int
		sixteenBytes := make([]byte, 16)
		n, err = h.File.Read(sixteenBytes)
		for n < 16 && err == nil {
			var i int
			i, err = h.File.Read(sixteenBytes[i:])
			n += i
		}

		outputBytes := []byte(hex.Dump(sixteenBytes))
		if h.bytesRead > 0 {
			nAsBytes := []byte(fmt.Sprintf("%0.8x", h.bytesRead))
			copy(outputBytes[0:8], nAsBytes)
		}

		copy(p[bytesInP:], outputBytes)
		if len(p[bytesInP:]) < len(outputBytes) {
			numLeftover := len(outputBytes) - len(p[bytesInP:])
			bytesInP += len(p[bytesInP:])
			leftover = make([]byte, len(outputBytes)-len(p[bytesInP:]))
			copy(leftover, outputBytes[len(outputBytes)-numLeftover:])
		} else {
			bytesInP += len(outputBytes)
		}
		h.bytesRead += n
	}
	h.buffer = leftover
	return bytesInP, err
}

type Verber struct {
	io.Reader
	io.Writer
	io.Seeker
}

type HexVerber struct {
	File   io.ReadWriteSeeker
	reader *HexReader
}

func NewHexVerber(f io.ReadWriteSeeker) HexVerber {
	h := HexVerber{
		File: f,
	}
	return h
}

func (h *HexVerber) Seek(offset int64, whence int) (int64, error) {
	return h.File.Seek(offset, whence)
}

func (h *HexVerber) Write(p []byte) (int, error) {
	n, err := h.File.Write(p)
	if err == nil || err == io.EOF {
		h.Seek(int64(-n), 1)
	}
	return n, err
}

func (h *HexVerber) Read(p []byte) (int, error) {
	r := NewHexReader(h.File)
	return r.Read(p)
}

func test() {
	flag.Parse()
	fname := flag.Arg(0)
	if fname == "" {
		fmt.Println("You must provide a filename as the first argument :(")
		return
	}

	file, err := os.Open(fname)
	if err != nil {
		fmt.Printf("Error opening: %s\n", err.Error())
		return
	}

	reader := NewHexReader(file)
	io.Copy(os.Stdout, &reader)

	file2, err := os.OpenFile(fname, os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println(`¯\_(ツ)_/¯  `)
		return
	}
	verber := NewHexVerber(file2)
	io.Copy(os.Stdout, &verber)

	n, err := verber.Seek(0, 0)
	fmt.Println(n, err)
	bytesToWrite := []byte("hi")
	i, err := verber.Write(bytesToWrite)
	fmt.Println(i, err)
	io.Copy(os.Stdout, &verber)
}

func main() {
	flag.Parse()
	fname := flag.Arg(0)
	if fname == "" {
		fmt.Println("You must provide a file name! :(")
		return
	}
	file, err := os.OpenFile(fname, os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println("error opening file: %s\n", err.Error())
		return
	}
	verber := NewHexVerber(file)
	stdin := bufio.NewReader(os.Stdin)
	for {
		verber.Seek(0, 0)
		io.Copy(os.Stdout, &verber)

		fmt.Println("")
		fmt.Print("Edit which line?  ")

		var line int64
		fmt.Scanln(&line)
		fmt.Println("Type in your new hex values here...")

		inputBytes, _, err := stdin.ReadLine()
		if err != nil {
			fmt.Println("error reading: %s", err.Error())
			continue
		}
		input := string(inputBytes)
		input = strings.Replace(input, " ", "", -1)
		runes := []rune(input)

		bytes := make([]byte, 0, len(runes)/2)
		for idx := 0; idx < len(runes); idx += 2 {
			pair := string(runes[idx : idx+2])
			byte64, err := strconv.ParseUint(pair, 16, 8)
			if err != nil {
				fmt.Printf("%v is not a valid byte: %s", pair, err.Error())
				break
			}
			bytes = append(bytes, byte(byte64))
		}

		verber.Seek(line, 0)
		verber.Write(bytes)
	}
}

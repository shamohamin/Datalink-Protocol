package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"

	structure "github.com/shamohamin/go-back-N-ARQ/framestructure"
)

// PORT for listening
const PORT = ":8001"

func handlingErr(err error) {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func main() {
	addr, err := net.ResolveTCPAddr("tcp", PORT)
	handlingErr(err)

	listener, err := net.ListenTCP("tcp", addr)
	handlingErr(err)

	var windowSize int

	if len(os.Args) == 1 {
		windowSize = 8
	} else {
		l, _ := strconv.ParseInt(os.Args[1], 10, 32)
		windowSize = int(l)
	}

	for {
		conn, err := listener.Accept()
		handlingErr(err)
		if err != nil {
			continue
		}
		go handlingConn(conn, int(windowSize))
	}
}

func handlingConn(conn net.Conn, windowSize int) {
	readerChan := make(chan []byte)
	writerChan := make(chan []byte)
	go connReader(conn, readerChan)
	go connWriter(conn, writerChan)

	// sending connected message
	writerChan <- []byte("*Connection Address(Client IP ADRRESS): " + conn.RemoteAddr().String() +
		"\r**Message: Welcom, connection to server stablished!" + "\n")

	// sending window size to client
	writerChan <- structure.NewInformationFrame(
		[]byte(strconv.Itoa(windowSize)), 0x0).MakeInformationByteFromFrame()

	goBackNARQ(conn, readerChan, writerChan, windowSize)
}

func goBackNARQ(conn net.Conn, readerChan, writerChan chan []byte, windowSize int) {
	aggrigatedBuffers := make([]*structure.FrameInformation, 0)
	var over bool = false
	// this use for timeout logic
	done := make(chan bool, 1)

	for {
		buffer := make([]*structure.Frame, windowSize)
		// recieving frames in max of window size
		i := 0
		for ; i < windowSize; i++ {
			// timeout logic
			go func() {
				timeout := time.Tick(structure.TIMEOUT)
				for {
					select {
					case <-done:
						return
					case <-timeout:
						readerChan <- nil
					}
				}
			}()
			// getting frame
			msg, ok := <-readerChan
			done <- true

			frame := structure.ParseFrameFromBytes(msg)
			// fmt.Println(frame)
			// if frame is damaged
			if frame == nil && i == windowSize-1 {
				break
			}
			// simulating the Frame lost
			if frame == nil || frame.FindTypeOfFrame() == structure.FrameLost {
				fmt.Println("frame nil")
				continue
			}
			// if its last frame or disconnet end listing
			if !ok || frame.FindTypeOfFrame() == structure.FrameDisconnect {
				over = true
				break
			}

			// buffer = append(buffer, frame)
			// save to buffer
			buffer[i] = frame
			// fmt.Println(frame, structure.Reset)
		}

		// if listening is over
		if over {
			writerChan <- structure.NewDisconnectFrame([]byte("EXIT"), 0x0).MakeInformationByteFromFrame()
			break
		}
		// logic for implementing proccessing the buffer for malformed or damaged frames
		buffer, err := checkingForFrameLost(buffer, writerChan, readerChan)
		buffer = sendingAck(buffer, err, readerChan, writerChan, windowSize)

		fmt.Println("completed")
		// add buffer to aggregated Buffer
		for j, f := range buffer {
			aggrigatedBuffers = append(aggrigatedBuffers,
				&structure.FrameInformation{
					FrameNumber:           int64(j),
					SequenceNumberOfFrame: f.Control[1],
					SendedFrame:           f,
				})
		}

	}

	// fmt.Println("len of aggregated is: ", len(aggrigatedBuffers))
}

// logic for checking the damaged Frames
func checkingForFrameLost(buffer []*structure.Frame, witerChan, readerChan chan []byte) ([]*structure.Frame, error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Error in checking for lost, err: ", err)
		}
	}()

	holdingBuffer := make([]*structure.Frame, 0)
	counter := 0
	for _, f := range buffer {
		if f == nil {
			continue
		}
		// if frame is out of sequence
		if int(f.Control[1]) != counter {
			return holdingBuffer, errors.New("frame " + string(counter) + " lost")
		}
		// fmt.Println(f)
		holdingBuffer = append(holdingBuffer, f)
		counter++

	}
	return holdingBuffer, nil
}

func sendingAck(buffer []*structure.Frame, err error, readerChan, writerChan chan []byte, maxWindowSize int) []*structure.Frame {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Error in sending Ack, err: ", err)
		}
	}()

	var frame *structure.Frame
	var bufferHandler []*structure.Frame
	for _, f := range buffer {
		bufferHandler = append(bufferHandler, f)
	}
	// if there is error
	// must send Acknowledgement with their lostedFrame number
	if err != nil {
		frame = structure.NewRejectFrame([]byte("SRJ "+strconv.Itoa(len(buffer))), uint(len(buffer)))
		fmt.Println(frame)
		by := frame.MakeInformationByteFromFrame()
		// fmt.Println(by)
		writerChan <- by
		// transmission logic with specific SReject frame
		fmt.Println("STARTING REQUESTING LOSTED FRAMES")
		fmt.Println(len(buffer))
		for i := len(buffer); i < maxWindowSize; i++ {
			msg, ok := <-readerChan
			f := structure.ParseFrameFromBytes(msg)
			if !ok || f.FindTypeOfFrame() == structure.FrameDisconnect {
				break
			}
			fmt.Println(f)
			bufferHandler = append(bufferHandler, f)
		}

		// fmt.Println(bufferHandler)
		fmt.Println("END REQUESTING LOSTED FRAMES NEW BUFFER CREATED.")
	} else {
		// if there was no error so send ack 0 to get another window size
		frame = structure.NewSupervisedFrame([]byte("RR"+strconv.Itoa(len(buffer)%maxWindowSize)), 0x0)
		writerChan <- frame.MakeInformationByteFromFrame()
	}
	return bufferHandler
}

// this function is used for reading from connection
func connReader(conn net.Conn, readerChan chan []byte) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		bi := scanner.Bytes()
		for i, b := range bi {
			if b == byte(0xFF) {
				bi[i] = byte(10)
			}
		}
		readerChan <- bi
	}
}

// this function is used for sending info to client from connection
func connWriter(conn net.Conn, writerChan chan []byte) {
	for msg := range writerChan {
		io.Copy(conn, bytes.NewBuffer(msg))
	}
}

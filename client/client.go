package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	structure "github.com/shamohamin/go-back-N-ARQ/framestructure"
)

// PORT is for accessing listen port
const PORT = ":8001"

func handlingError(err error) {
	if err != nil {
		fmt.Printf("err is : %s", err.Error())
		os.Exit(1)
	}
}

// initializing random for making string from variables
var alphabet [26 * 2]int
var randGenerator *rand.Rand

func init() {
	randGenerator = rand.New(rand.NewSource(time.Now().Unix()))
}

func main() {
	for i := 0; i < 26; i++ {
		alphabet[i] = 'A' + i
		alphabet[i+26] = 'a' + i
	}
	addr, err := net.ResolveTCPAddr("tcp", PORT)
	handlingError(err)
	conn, err := net.DialTCP("tcp", nil, addr)
	handlingError(err)

	var maxFrame int
	if len(os.Args) == 1 {
		maxFrame = 10
	} else {
		maxFrame, _ = strconv.Atoi(os.Args[1])
	}
	handlingConnection(conn, maxFrame)
}

func handlingConnection(conn net.Conn, frameLen int) {
	readerChan := make(chan []byte)
	writerChan := make(chan []byte)
	go connReader(conn, readerChan)
	go connWriter(conn, writerChan)

	defer func() {
		if err := recover(); err != nil {
			fmt.Println("err occured in handling connection ", err)
		}
		conn.Close()
		close(readerChan)
		close(writerChan)
	}()
	// getting greeting message from server
	fmt.Println(string(<-readerChan))
	str, _ := <-readerChan
	// getting window size from server
	f := structure.ParseFrameFromBytes(str)
	windowSize, err := strconv.ParseInt(string(f.Information), 10, 32)
	fmt.Println("Frame For Getting:\n", f, "\nWindow Size From Server Is:", windowSize)
	// if we cant get window size we will set it to the 8
	if err != nil {
		windowSize = 8
	}
	// make random frames
	frames := makeFrames(int(windowSize), frameLen)
	sendTimes := make([]time.Duration, 0)

	counter := 0
	for counter < len(frames) {
		// sending window size
		start := time.Now()
		for i := counter; i < (int(windowSize)+counter) && i < len(frames); i++ {
			frame := frames[i]

			if frame == nil {
				writerChan <- []byte{126, 10}
				continue
			}
			// fmt.Println(frame.SendedFrame, structure.Reset)
			// sending frame
			writerChan <- frame.SendedFrame.MakeInformationByteFromFrame()
			// time.Sleep(1000 * time.Millisecond)
		}

		// Waiting For Acknowledgement
		ack, ok := <-readerChan
		if !ok {
			break
		}

		// wating for Ack
		f := structure.ParseFrameFromBytes(ack)
		t := f.FindTypeOfFrame()

		switch t {
		case structure.FrameReject: // if it's so it must retransmit
			fmt.Println(f, structure.Reset)
			handlingRetransmitionOfTheRejectFrame(f, frames, counter, int(windowSize), writerChan)
			fmt.Println("retransmistion completed")
		case structure.FrameSupervised: // if it was good and no error occured then it will get supervised frame and send next window size
			fmt.Println(f)
			fmt.Println("Starting sending next windows size.", structure.Reset)
		case structure.FrameDisconnect:
			fmt.Println(f)
		default:
			break
		}

		sendTimes = append(sendTimes, time.Since(start))
		counter += int(windowSize)
	}
	res := 0 * time.Millisecond
	for _, t := range sendTimes {
		res += t
	}

	// fmt.Println(res)
	writingFile(int(frameLen), res)
}

// for writing time of transmision
func writingFile(windowSize int, res time.Duration) {
	str := "Frame len, Time To send All Frames"
	str += "\n" + strconv.Itoa(windowSize) + ", " + res.String()
	fileName := "output.txt"
	if exits(fileName) {
		os.Remove(fileName)
	}
	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println(err)
	}
	f.Write([]byte(str))
}

// for cheking folder or file is exits or not
func exits(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// it used for handling retransmition of damaged frame with specific SReject frame
func handlingRetransmitionOfTheRejectFrame(
	frame *structure.Frame,
	frames []*structure.FrameInformation,
	counter, windowSize int,
	writerChan chan []byte) {

	fmt.Println(structure.Reset, "STARTING THE RETRANSMISION FROM LOSTED FRAME")
	from := int(frame.Control[1])
	// send from Srejec sequence number until the window size
	for i := from + counter; i < windowSize+counter; i++ {
		if frames[i] == nil {
			a := structure.NewInformationFrame([]byte("Frame "+strconv.Itoa(i)), uint(i%windowSize))
			fmt.Println(a)
			writerChan <- a.MakeInformationByteFromFrame()
			continue
		}
		writerChan <- frames[i].SendedFrame.MakeInformationByteFromFrame()
		fmt.Println(frames[i].SendedFrame)
		if frames[i].SendedFrame.FindTypeOfFrame() == structure.FrameDisconnect {
			break
		}
	}
	fmt.Println("ENDING THE RETRANSMISION FROM LOSTED FRAME", structure.Reset)
}

// make random frames for sending the frames to sever from client
func makeFrames(windowSize, frameLen int) []*structure.FrameInformation {
	buffer := make([]*structure.FrameInformation, 0)
	i := 0
	// buffer = append(buffer, &structure.FrameInformation{
	// 	SendedFrame: structure.NewLostFrame([]byte("Lost"), 0x0),
	// })
	for i = 0; i < int(structure.MAXFRAMECOUNT)-1; i++ {
		var temp *structure.FrameInformation
		// if i == 9 {
		// 	temp = nil
		// } else {
		temp = &structure.FrameInformation{
			FrameNumber:           int64(i),
			SequenceNumberOfFrame: byte(i % windowSize),
			SendedFrame: structure.NewInformationFrame(
				informationConstructor(frameLen, i), uint(i%windowSize)),
		}
		// }
		buffer = append(buffer, temp)
	}

	buffer = append(buffer, &structure.FrameInformation{
		FrameNumber:           int64(i),
		SequenceNumberOfFrame: byte(i % windowSize),
		SendedFrame: structure.NewInformationFrame(
			informationConstructor(frameLen, i), uint(i%windowSize)),
	})
	buffer = append(buffer, &structure.FrameInformation{
		FrameNumber:           int64(i),
		SequenceNumberOfFrame: byte(i % windowSize),
		SendedFrame: structure.NewInformationFrame(
			informationConstructor(frameLen, i+1), uint((i+1)%windowSize)),
	})
	// appending disconnect frame
	buffer = append(buffer, &structure.FrameInformation{
		SendedFrame: structure.NewDisconnectFrame([]byte("EXIT"), 0x0),
	})
	return buffer
}

// generate dummy string from alphabet for specific len for sending to server
func informationConstructor(len, FrameNumber int) []byte {
	buf := ""
	for i := 0; i < len; i++ {
		buf += string(alphabet[randGenerator.Intn(2*26)])
	}
	str := strconv.Itoa(FrameNumber)
	buf += " Frame " + string(str)
	return []byte(buf)
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
		str := string(bi)
		str = strings.ReplaceAll(str, "\r", "\n")
		readerChan <- []byte(str)
	}
}

// this function is used for sending info to client
func connWriter(conn net.Conn, writerChan chan []byte) {
	for msg := range writerChan {
		io.Copy(conn, bytes.NewBuffer(msg))
	}
}

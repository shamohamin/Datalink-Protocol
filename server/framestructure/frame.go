package framestructure

import (
	"fmt"
	"time"
)

// Frame is HDLC frame structure
type Frame struct {
	Information []byte
	StartFlag   byte
	EndFlag     byte
	Control     [2]byte
}

// FrameInformation is for getting sending frame Information
type FrameInformation struct {
	SendedFrame                  *Frame
	AcknowledgementTimeReciveced time.Duration
	SequenceNumberOfFrame        byte
	FrameNumber                  int64
	NumberOfRetransmition        int
}

// FrameType is a registered frame type as defined in HDLC protocol
type FrameType uint8

// FrameData: is I-Frame
// FrameSupervised: is S-Frame
// FrameLost: for Simulating Losted Frame for go-back-N-ARQ
// FrameReject: for rejecttion and
const (
	FrameData       FrameType = 0x0
	FrameSupervised FrameType = 0x2
	FrameReject     FrameType = 0x3
	FrameLost       FrameType = 0x4
	FrameMalFormed  FrameType = 0x5
	FrameDisconnect FrameType = 0x6
)

// MAXFRAMECOUNT : maximum frame count for sending to server from client
// TIMEOUT: maximum time for retransmition of the frame initialValue is 1.5 seconds
// MAXRETRANSMITION: maximum number of retransmition for frame
const (
	MAXFRAMECOUNT        uint          = 67
	TIMEOUT              time.Duration = 1*time.Second + 500*time.Millisecond
	MAXRETRANSMITION     uint          = 2
	FrameHeaderLen       uint          = 0x18 // length of frame header to send
	FrameTrailerLen      uint          = 0x8
	StartByte            uint          = 0b01111110
	EndByte              uint          = 0b01111110
	StartBytePortion     uint          = 0
	FrameTypePortion     uint          = 1
	FrameSequencePortion uint          = 2
)

// MakeInformationByteFromFrame make byte frame from values in frame struct
func (f *Frame) MakeInformationByteFromFrame() []byte {
	generatedByteFromFrame := make([]byte, 0)
	// Appending startFlag
	generatedByteFromFrame = append(generatedByteFromFrame, f.StartFlag)
	// seting control flags (sequence number and frame type)
	for _, c := range f.Control {
		generatedByteFromFrame = append(generatedByteFromFrame, c)
	}
	// Appending information
	for _, info := range f.Information {
		generatedByteFromFrame = append(generatedByteFromFrame, info)
	}
	// appending endflags
	generatedByteFromFrame = append(generatedByteFromFrame, f.EndFlag)

	// checking for seqence of end flag if it is then replace it with someThing else
	for i, ib := range generatedByteFromFrame {
		if ib == byte(10) {
			generatedByteFromFrame[i] = byte(0xFF)
		}
	}
	// append '\n' for scanner can detect this frame
	generatedByteFromFrame = append(generatedByteFromFrame, byte(10))

	return generatedByteFromFrame
}

// NewInformationFrame creates new Information Frame
func NewInformationFrame(info []byte, sequenceNumber uint) *Frame {
	frame := new(Frame)
	frame.StartFlag = byte(StartByte)
	frame.Control[0] = byte(FrameData)
	frame.Control[1] = byte(sequenceNumber)
	frame.Information = info
	frame.EndFlag = byte(EndByte)
	return frame
}

// NewSupervisedFrame creates new Supervised Frame
func NewSupervisedFrame(info []byte, sequenceNumber uint) *Frame {
	frame := new(Frame)
	frame.StartFlag = byte(StartByte)
	frame.Control[0] = byte(FrameSupervised)
	frame.Control[1] = byte(sequenceNumber)
	frame.Information = info
	frame.EndFlag = byte(EndByte)
	return frame
}

// NewRejectFrame is for making rejected frame for go back-N-ARQ
func NewRejectFrame(info []byte, sequenceNumber uint) *Frame {
	frame := new(Frame)
	frame.StartFlag = byte(StartByte)
	frame.Control[0] = byte(FrameReject)
	frame.Control[1] = byte(sequenceNumber)
	frame.Information = info
	frame.EndFlag = byte(EndByte)
	return frame
}

// NewDisconnectFrame is for making rejected frame for go back-N-ARQ
func NewDisconnectFrame(info []byte, sequenceNumber uint) *Frame {
	frame := new(Frame)
	frame.StartFlag = byte(StartByte)
	frame.Control[0] = byte(FrameDisconnect)
	frame.Control[1] = byte(sequenceNumber)
	frame.Information = info
	frame.EndFlag = byte(EndByte)
	return frame
}

// NewLostFrame is for making lostFrame simulation
func NewLostFrame(info []byte, sequenceNumber uint) *Frame {
	frame := new(Frame)
	frame.StartFlag = byte(StartByte)
	frame.Control[0] = byte(FrameLost)
	frame.Control[1] = byte(sequenceNumber)
	frame.Information = info
	frame.EndFlag = byte(EndByte)
	return frame
}

// ParseFrameFromBytes make Frame from bytes
func ParseFrameFromBytes(frame []byte) *Frame {
	f := new(Frame)
	i := 0
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("error in parsing")
		}
	}()
	// fmt.Println("in bit:", frame)
	for ; i < int(FrameHeaderLen)/8; i++ {
		switch uint(i) {
		case StartBytePortion:
			f.StartFlag = frame[i]
		case FrameTypePortion:
			f.Control[0] = frame[i]
		case FrameSequencePortion:
			f.Control[1] = frame[i]
		default:
			break
		}
	}
	buff := make([]byte, 0)
	for j := i; j < len(frame)-1; j++ {
		buff = append(buff, frame[j])
	}

	f.Information = buff
	f.EndFlag = frame[len(frame)-1]

	return f
}

// FindTypeOfFrame is utility function for distinguiting of Frame Type
func (f *Frame) FindTypeOfFrame() FrameType {
	switch FrameType(f.Control[0]) {
	case FrameSupervised:
		return FrameSupervised
	case FrameData:
		return FrameData
	case FrameLost:
		return FrameLost
	case FrameReject:
		return FrameReject
	case FrameDisconnect:
		return FrameDisconnect
	default:
		return FrameMalFormed
	}
}

// Colors for the terminal
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
)

func (f Frame) String() string {
	if f.FindTypeOfFrame() == FrameData {
		fmt.Println(Green)
	} else if f.FindTypeOfFrame() == FrameSupervised {
		fmt.Println(Yellow)
	} else if f.FindTypeOfFrame() == FrameLost || f.FindTypeOfFrame() == FrameReject {
		fmt.Println(Red)
	}

	str := "****************************************** \n"
	// str += fmt.Sprintf("Frame Information in bytes: \n%b \n", f.Information)
	str += fmt.Sprintf("Frame Information in string: \"%s\" \n", string(f.Information))

	switch f.FindTypeOfFrame() {
	case FrameData:
		str += "Frame Type: Information Frame \n"
	case FrameSupervised:
		str += "Frame Type: Supervised Frame \n"
	case FrameDisconnect:
		str += "Frame Type: Disconnect Frame \n"
	case FrameReject:
		str += "Frame Type: Reject Frame \n"
	default:
	}

	str += fmt.Sprintf("Frame Sequence: '\\x%02x' '\\b%b' \n", f.Control[1], f.Control[1])
	str += "******************************************"
	return str
}

package main

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"time"

	"github.com/nonoo/kappanhang/log"
)

type controlStream struct {
	common           streamCommon
	authSendSeq      uint16
	authInnerSendSeq uint16
	authID           [6]byte

	serialAndAudioStreamOpened   bool
	requestSerialAndAudioTimeout *time.Timer
}

func (s *controlStream) sendPktAuth() {
	// The reply to the auth packet will contain a 6 bytes long auth ID with the first 2 bytes set to our randID.
	var randID [2]byte
	_, err := rand.Read(randID[:])
	if err != nil {
		exit(err)
	}
	p := []byte{0x80, 0x00, 0x00, 0x00, 0x00, 0x00, byte(s.authSendSeq), byte(s.authSendSeq >> 8),
		byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
		byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID),
		0x00, 0x00, 0x00, 0x70, 0x01, 0x00, 0x00, byte(s.authInnerSendSeq),
		byte(s.authInnerSendSeq >> 8), 0x00, randID[0], randID[1], 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x2b, 0x3f, 0x55, 0x5c, 0x00, 0x00, 0x00, 0x00, // username: beer
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x2b, 0x3f, 0x55, 0x5c, 0x3f, 0x25, 0x77, 0x58, // pass: beerbeer
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x69, 0x63, 0x6f, 0x6d, 0x2d, 0x70, 0x63, 0x00, // icom-pc in plain text
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	s.common.send(p)
	s.common.send(p)

	s.authSendSeq++
	s.authInnerSendSeq++
}

func (s *controlStream) sendPktReauth(firstReauthSend bool) {
	var magic byte

	if firstReauthSend {
		magic = 0x02
	} else {
		magic = 0x05
	}

	// Example request from PC:  0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0d, 0x00,
	//                           0xbb, 0x41, 0x3f, 0x2b, 0xe6, 0xb2, 0x7b, 0x7b,
	//                           0x00, 0x00, 0x00, 0x30, 0x01, 0x05, 0x00, 0x02,
	//                           0x00, 0x00, 0x5d, 0x37, 0x12, 0x82, 0x3b, 0xde,
	//                           0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	//                           0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	//                           0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	//                           0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00
	// Example reply from radio: 0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0e, 0x00,
	//                           0xe6, 0xb2, 0x7b, 0x7b, 0xbb, 0x41, 0x3f, 0x2b,
	//                           0x00, 0x00, 0x00, 0x30, 0x02, 0x05, 0x00, 0x02,
	//                           0x00, 0x00, 0x5d, 0x37, 0x12, 0x82, 0x3b, 0xde,
	//                           0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	//                           0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	//                           0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	//                           0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00
	p := []byte{0x40, 0x00, 0x00, 0x00, 0x00, 0x00, byte(s.authSendSeq), byte(s.authSendSeq >> 8),
		byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
		byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID),
		0x00, 0x00, 0x00, 0x30, 0x01, magic, 0x00, byte(s.authInnerSendSeq),
		byte(s.authInnerSendSeq >> 8), 0x00, s.authID[0], s.authID[1], s.authID[2], s.authID[3], s.authID[4], s.authID[5],
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	s.common.send(p)
	s.common.send(p)

	s.authSendSeq++
	s.authInnerSendSeq++
}

func (s *controlStream) sendDisconnect() {
	if s.common.conn == nil {
		return
	}
	// s.common.send([]byte{0x40, 0x00, 0x00, 0x00, 0x00, 0x00, byte(s.authSendSeq), byte(s.authSendSeq >> 8),
	// 	byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
	// 	byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID),
	// 	0x00, 0x00, 0x00, 0x30, 0x01, 0x01, 0x00, byte(s.authInnerSendSeq),
	// 	byte(s.authInnerSendSeq >> 8), 0x00, s.authID[0], s.authID[1], s.authID[2], s.authID[3], s.authID[4], s.authID[5],
	// 	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	// 	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	// 	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	// 	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	s.common.sendDisconnect()
}

func (s *controlStream) sendPkt0() {
	p := []byte{0x10, 0x00, 0x00, 0x00, 0x00, 0x00, byte(s.authSendSeq), byte(s.authSendSeq >> 8),
		byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
		byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID)}
	s.common.send(p)
	s.common.send(p)

	s.authSendSeq++
}

func (s *controlStream) sendRequestSerialAndAudio() {
	log.Print("requesting serial and audio stream")
	p := []byte{0x90, 0x00, 0x00, 0x00, 0x00, 0x00, byte(s.authSendSeq), byte(s.authSendSeq >> 8),
		byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
		byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID),
		0x00, 0x00, 0x00, 0x80, 0x01, 0x03, 0x00, byte(s.authInnerSendSeq),
		byte(s.authInnerSendSeq >> 8), 0x00, s.authID[0], s.authID[1], s.authID[2], s.authID[3], s.authID[4], s.authID[5],
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10,
		0x80, 0x00, 0x00, 0x90, 0xc7, 0x0e, 0x86, 0x01, // The last 5 bytes from this row can be acquired from a reply starting with 0xa8 or 0x90
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x49, 0x43, 0x2d, 0x37, 0x30, 0x35, 0x00, 0x00, // IC-705 in plain text
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x2b, 0x3f, 0x55, 0x5c, 0x00, 0x00, 0x00, 0x00, // username: beer
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x01, 0x01, 0x04, 0x04, 0x00, 0x00, 0xbb, 0x80,
		0x00, 0x00, 0xbb, 0x80, 0x00, 0x00, 0xc3, 0x52,
		0x00, 0x00, 0xc3, 0x53, 0x00, 0x00, 0x00, 0xa0,
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	s.common.send(p)
	s.common.send(p)

	s.authSendSeq++
	s.authInnerSendSeq++

	s.requestSerialAndAudioTimeout = time.AfterFunc(3*time.Second, func() {
		exit(errors.New("serial and audio request timeout"))
	})
}

func (s *controlStream) handleRead(r []byte) {
	switch len(r) {
	case 16:
		if bytes.Equal(r[:6], []byte{0x10, 0x00, 0x00, 0x00, 0x00, 0x00}) {
			// Replying to the radio.
			// Example request from radio: 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x13, 0x00, 0xe4, 0x35, 0xdd, 0x72, 0xbe, 0xd9, 0xf2, 0x63
			// Example answer from PC:     0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x13, 0x00, 0xbe, 0xd9, 0xf2, 0x63, 0xe4, 0x35, 0xdd, 0x72
			gotSeq := binary.LittleEndian.Uint16(r[6:8])
			p := []byte{0x10, 0x00, 0x00, 0x00, 0x00, 0x00, byte(gotSeq), byte(gotSeq >> 8),
				byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
				byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID)}
			s.common.send(p)
			s.common.send(p)
		}
	case 80:
		if bytes.Equal(r[:6], []byte{0x50, 0x00, 0x00, 0x00, 0x00, 0x00}) && bytes.Equal(r[48:51], []byte{0xff, 0xff, 0xff}) {
			// Example answer from radio: 0x50, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x00,
			//							  0x86, 0x1f, 0x2f, 0xcc, 0x03, 0x03, 0x89, 0x29,
			//							  0x00, 0x00, 0x00, 0x40, 0x02, 0x03, 0x00, 0x52,
			//							  0x00, 0x00, 0xf8, 0xad, 0x06, 0x8d, 0xda, 0x7b,
			//							  0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10,
			//							  0x80, 0x00, 0x00, 0x90, 0xc7, 0x0e, 0x86, 0x01,
			//							  0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00,
			//							  0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			//							  0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			//							  0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00

			exit(errors.New("reauth failed, try again after about 1 minute"))
		}
	case 144:
		if !s.serialAndAudioStreamOpened && bytes.Equal(r[:6], []byte{0x90, 0x00, 0x00, 0x00, 0x00, 0x00}) && r[96] == 1 {
			// Example answer:
			// 0x90, 0x00, 0x00, 0x00, 0x00, 0x00, 0x19, 0x00,
			// 0xc6, 0x5f, 0x6f, 0x0c, 0x5f, 0x8b, 0x1e, 0x89,
			// 0x00, 0x00, 0x00, 0x80, 0x03, 0x00, 0x00, 0x00,
			// 0x00, 0x00, 0x31, 0x30, 0x31, 0x47, 0x39, 0x07,
			// 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10,
			// 0x80, 0x00, 0x00, 0x90, 0xc7, 0x0e, 0x86, 0x01,
			// 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			// 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			// 0x49, 0x43, 0x2d, 0x37, 0x30, 0x35, 0x00, 0x00,
			// 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			// 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			// 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			// 0x01, 0x00, 0x00, 0x00, 0x69, 0x63, 0x6f, 0x6d,
			// 0x2d, 0x70, 0x63, 0x00, 0x00, 0x00, 0x00, 0x00,
			// 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			// 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			// 0x00, 0x00, 0x00, 0x00, 0xc0, 0xa8, 0x03, 0x03,
			// 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00
			log.Print("serial and audio request success")
			if s.requestSerialAndAudioTimeout != nil {
				s.requestSerialAndAudioTimeout.Stop()
				s.requestSerialAndAudioTimeout = nil
			}
			go streams.serial.start()
			go streams.audio.start()
			s.serialAndAudioStreamOpened = true
		}
	}
}

func (s *controlStream) init() {
	s.common.open("control", 50001)
}

func (s *controlStream) start() {
	startTime := time.Now()

	s.common.sendPkt3()
	s.common.pkt7.sendSeq = 1
	s.common.pkt7.send(&s.common)
	s.common.sendPkt3()
	s.common.waitForPkt4Answer()
	s.common.sendPkt6()
	s.common.waitForPkt6Answer()

	s.authSendSeq = 1
	s.authInnerSendSeq = 1
	s.sendPktAuth()

	log.Debug("expecting auth answer")
	// Example success auth packet: 0x60, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00,
	//                              0xe6, 0xb2, 0x7b, 0x7b, 0xbb, 0x41, 0x3f, 0x2b,
	//                              0x00, 0x00, 0x00, 0x50, 0x02, 0x00, 0x00, 0x00,
	//                              0x00, 0x00, 0x5d, 0x37, 0x12, 0x82, 0x3b, 0xde,
	//                              0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	//                              0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	//                              0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	//                              0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	//                              0x46, 0x54, 0x54, 0x48, 0x00, 0x00, 0x00, 0x00,
	//                              0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	//                              0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	//                              0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00
	r := s.common.expect(96, []byte{0x60, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00})
	if bytes.Equal(r[48:52], []byte{0xff, 0xff, 0xff, 0xfe}) {
		exit(errors.New("invalid user/password"))
	}

	copy(s.authID[:], r[26:32])
	log.Print("auth ok, waiting a bit")

	time.AfterFunc(1*time.Second, func() {
		log.Print("sending reauth 1/2...")
		s.sendPktReauth(true)
		time.AfterFunc(1*time.Second, func() {
			log.Print("sending reauth 2/2...")
			s.sendPktReauth(false)
			time.AfterFunc(time.Second, func() {
				s.sendRequestSerialAndAudio()
			})
		})
	})

	s.common.pkt7.startPeriodicSend(&s.common, 5, false)

	pkt0SendTicker := time.NewTicker(100 * time.Millisecond)
	reauthTicker := time.NewTicker(60 * time.Second)
	statusLogTicker := time.NewTicker(3 * time.Second)

	for {
		select {
		case r = <-s.common.readChan:
			s.handleRead(r)
		case <-pkt0SendTicker.C:
			s.sendPkt0()
		case <-reauthTicker.C:
			s.sendPktReauth(false)
		case <-statusLogTicker.C:
			if s.serialAndAudioStreamOpened {
				log.Print("running for ", time.Since(startTime), " roundtrip latency ", s.common.pkt7.latency)
			}
		}
	}
}

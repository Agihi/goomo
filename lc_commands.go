package goomo

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type CSSTCommand struct {
	Port   string
	Stream string
}

func (c *CSSTCommand) Tag() CommandTag {
	return CSST
}

func (c *CSSTCommand) MsgFormat() ([]byte, error) {
	req := make([]byte, 15)
	copy(req[0:4], "CSST")
	binary.BigEndian.PutUint32(req[4:8], 15)
	portInt, err := strconv.Atoi(strings.TrimPrefix(c.Port, ":"))
	if err != nil {
		return nil, fmt.Errorf("port conversion to int: %v", err)
	}
	binary.BigEndian.PutUint32(req[8:12], uint32(portInt))
	copy(req[12:15], c.Stream)
	return req, nil
}

type CESTCommand struct {
	Port   string
	Stream string
}

func (c *CESTCommand) Tag() CommandTag {
	return CEST
}

func (c *CESTCommand) MsgFormat() ([]byte, error) {
	req := make([]byte, 15)
	copy(req[0:4], "CEST")
	binary.BigEndian.PutUint32(req[4:8], 15)
	portInt, err := strconv.Atoi(strings.TrimPrefix(c.Port, ":"))
	if err != nil {
		return nil, fmt.Errorf("port conversion to int: %v", err)
	}
	binary.BigEndian.PutUint32(req[8:12], uint32(portInt))
	copy(req[12:15], c.Stream)
	return req, nil
}

type CSPSCommand struct {
	X float32
	Y float32
}

func (c *CSPSCommand) Tag() CommandTag {
	return CSPS
}

func (c *CSPSCommand) MsgFormat() ([]byte, error) {
	req := make([]byte, 16)
	copy(req[0:4], "CSPS")
	binary.BigEndian.PutUint32(req[4:8], 16)
	binary.BigEndian.PutUint32(req[8:12], math.Float32bits(c.X))
	binary.BigEndian.PutUint32(req[12:16], math.Float32bits(c.Y))
	return req, nil
}

type CLVLCommand struct {
	Lv float32
}

func (c *CLVLCommand) Tag() CommandTag {
	return CLVL
}

func (c *CLVLCommand) MsgFormat() ([]byte, error) {
	req := make([]byte, 12)
	copy(req[0:4], "CLVL")
	binary.BigEndian.PutUint32(req[4:8], 12)
	binary.BigEndian.PutUint32(req[8:12], math.Float32bits(c.Lv))
	return req, nil
}

type CAVLCommand struct {
	Av float32
}

func (c *CAVLCommand) Tag() CommandTag {
	return CAVL
}

func (c *CAVLCommand) MsgFormat() ([]byte, error) {
	req := make([]byte, 12)
	copy(req[0:4], "CAVL")
	binary.BigEndian.PutUint32(req[4:8], 12)
	binary.BigEndian.PutUint32(req[8:12], math.Float32bits(c.Av))
	return req, nil
}

type CMHDCommand struct {
	vert float32
	hori float32
}

package ignite

import (
	"context"
	"errors"
	"fmt"
	"github.com/cockroachdb/apd/v3"
	"reflect"
)

const (
	opResourceClose int16 = 0
)

type Cursor interface {
	Close() error
	Scan(src ...any) error
	Next() bool
	Err() error
	Columns() []string
}

type baseCursor struct {
	ctx        context.Context
	ch         channel
	marsh      marshaller
	curId      int64
	currIdx    int // must be set to negative on creation (before first iteration)
	rowSize    int
	opPageNext int16
	data       []interface{}
	hasMore    bool
	lastErr    error
}

func (c *baseCursor) Next() bool {
	if c.lastErr != nil {
		return false
	}
	// set current idx to zero on the first iteration
	if c.currIdx < 0 {
		c.currIdx = 0
	} else {
		c.currIdx += c.rowSize
	}
	if c.currIdx < len(c.data) {
		return true
	}
	if !c.hasMore {
		return false
	}
	c.ch.send(c.ctx, c.opPageNext,
		func(_ channel, output BinaryOutputStream) error {
			output.WriteInt64(c.curId)
			return nil
		},
		func(_ channel, input BinaryInputStream, err error) {
			if err != nil {
				c.lastErr = err
			}
			err = c.readData(input)
			if err != nil {
				c.lastErr = err
			}
			c.currIdx = 0
		})
	if c.lastErr != nil {
		return false
	}
	return c.currIdx < len(c.data)
}

func (c *baseCursor) Err() error {
	return c.lastErr
}

func (c *baseCursor) readData(input BinaryInputStream) error {
	sz := int(input.ReadInt32()) * c.rowSize
	c.data = make([]interface{}, 0, sz)
	err := readSequence(input, sz, func(_ int, reader BinaryInputStream) error {
		val, err0 := c.marsh.unmarshal(c.ctx, reader)
		if err0 != nil {
			return err0
		}
		c.data = append(c.data, val)
		return nil
	})
	if err != nil {
		return err
	}
	c.hasMore = input.ReadBool()
	return nil
}

func (c *baseCursor) Close() error {
	if !c.hasMore {
		return nil
	}
	defer func() {
		c.hasMore = false
	}()
	var err error
	c.ch.send(c.ctx, opResourceClose, func(_ channel, output BinaryOutputStream) error {
		output.WriteInt64(c.curId)
		return nil
	}, func(_ channel, input BinaryInputStream, err0 error) {
		if err0 != nil {
			err = err0
			c.lastErr = err0
		}
	})
	return err
}

func (c *baseCursor) Scan(src ...any) error {
	if len(src) < c.rowSize {
		return fmt.Errorf("number of parameters %d less than row size %d", len(src), c.rowSize)
	}
	for i := 0; i < len(src); i++ {
		if err := convertAssign(src[i], c.data[c.currIdx+i]); err != nil {
			return err
		}
	}
	return nil
}

func convertAssign(dest interface{}, src interface{}) error {
	dpv := reflect.ValueOf(dest)
	if dpv.Kind() != reflect.Pointer {
		return errors.New("destination is not a pointer")
	}
	if dpv.IsNil() {
		return errors.New("destination is nil")
	}
	// decimal case:
	switch destDec := dest.(type) {
	case *apd.Decimal:
		switch srcDec := src.(type) {
		case *apd.Decimal:
			*destDec = *srcDec
			return nil
		case nil:
			*destDec = apd.Decimal{}
			return nil
		}
	}
	dv := reflect.Indirect(dpv)
	if src == nil {
		dv.Set(reflect.Zero(dv.Type()))
		return nil
	}
	sv := reflect.ValueOf(src)
	if sv.IsValid() && sv.Type().AssignableTo(dv.Type()) {
		dv.Set(sv)
		return nil
	}
	if dv.Kind() == reflect.Pointer {
		dv.Set(reflect.New(dv.Type().Elem()))
		return convertAssign(dv.Interface(), src)
	}
	return fmt.Errorf("cannot assign %T(%v) to %v", src, src, dv.Type())
}

package ignite

import "time"

type Time int64

type Date int64

func NewTime(val time.Time) Time {
	utcVal := val.UTC()
	return Time(utcVal.Unix()*1000 + int64(utcVal.Nanosecond())/int64(time.Millisecond))
}

func (t Time) Time() time.Time {
	return time.Unix(int64(t)/1000, (int64(t)%1000)*int64(time.Millisecond))
}

func (t Time) String() string {
	return t.Time().Format("15:04:05.000")
}

func NewDate(val time.Time) Date {
	utcVal := val.UTC()
	return Date(utcVal.Unix()*1000 + int64(utcVal.Nanosecond())/int64(time.Millisecond))
}

func (d Date) Time() time.Time {
	return time.Unix(int64(d)/1000, (int64(d)%1000)*int64(time.Millisecond))
}

func (d Date) String() string {
	return d.Time().String()
}

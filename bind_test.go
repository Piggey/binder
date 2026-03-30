package binder

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type (
	TestHeader struct {
		MyStr   string    `header:"My-Str"`
		MyInt   int       `header:"My-Int"`
		MyTime  time.Time `header:"My-Time"`
		MySlice []float64 `header:"My-Slice"`
	}

	TestQuery struct {
		MyStr   string    `query:"myStr"`
		MyInt   int       `query:"myInt"`
		MyTime  time.Time `query:"myTime"`
		MySlice []float64 `query:"mySlice"`
	}
)

func Test_bindReflect(t *testing.T) {
	type args struct {
		r   *http.Request
		obj any
	}
	tests := []struct {
		name    string
		args    args
		want    any
		wantErr bool
	}{
		{
			name: "not a pointer to struct -> return error",
			args: args{
				r:   &http.Request{},
				obj: new(int),
			},
			want:    new(int),
			wantErr: true,
		},
		{
			name: "slice instead of struct ptr -> return error",
			args: args{
				r:   &http.Request{},
				obj: []TestHeader{},
			},
			want:    []TestHeader{},
			wantErr: true,
		},
		{
			name: "nil -> return error",
			args: args{
				r:   &http.Request{},
				obj: nil,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "parse header",
			args: args{
				r: &http.Request{
					Header: http.Header{
						"My-Str":   []string{"hey"},
						"My-Int":   []string{"69"},
						"My-Time":  []string{"2026-03-30T21:15:00Z"},
						"My-Slice": []string{"3.14", "1.618"},
					},
				},
				obj: new(TestHeader),
			},
			want: &TestHeader{
				MyStr:   "hey",
				MyInt:   69,
				MyTime:  time.Date(2026, 03, 30, 21, 15, 0, 0, time.UTC),
				MySlice: []float64{3.14, 1.618},
			},
		},
		{
			name: "parse query",
			args: args{
				r: &http.Request{
					URL: &url.URL{
						RawQuery: "myStr=hey&myInt=69&myTime=2026-03-30T21:15:00Z&mySlice=3.14&mySlice=1.618",
					},
				},
				obj: new(TestQuery),
			},
			want: &TestQuery{
				MyStr:   "hey",
				MyInt:   69,
				MyTime:  time.Date(2026, 03, 30, 21, 15, 0, 0, time.UTC),
				MySlice: []float64{3.14, 1.618},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bindReflect(tt.args.r, tt.args.obj)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.want, tt.args.obj)
		})
	}
}

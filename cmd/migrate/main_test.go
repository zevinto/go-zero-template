package main

import "testing"

func TestPositiveVersion(t *testing.T) {
	cases := []struct {
		in      string
		want    uint
		wantErr bool
	}{
		{in: "3", want: 3},
		{in: "0004", want: 4},      // 允许前导零，与迁移文件 NNNN 序号风格一致
		{in: "0", wantErr: true},   // 版本必须为正
		{in: "-1", wantErr: true},  // 负数交给 force，不用 migrate
		{in: "abc", wantErr: true}, // 非数字
		{in: "", wantErr: true},    // 空
		{in: "1.5", wantErr: true}, // 非整数
	}
	for _, c := range cases {
		got, err := positiveVersion(c.in)
		if c.wantErr {
			if err == nil {
				t.Fatalf("positiveVersion(%q) error = nil, want error", c.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("positiveVersion(%q) error = %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("positiveVersion(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

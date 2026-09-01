package xutil

import "testing"

func TestDedent(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "剥公共前导 tab 缩进、去首尾空行",
			in: `第一行
		第二行
		第三行`,
			want: "第一行\n第二行\n第三行",
		},
		{
			name: "保留相对缩进（第二行多一层）",
			in: `首段
			首段相对内容
			  子项（再深一层）`,
			want: "首段\n首段相对内容\n  子项（再深一层）",
		},
		{
			name: "空行保留为段落分隔",
			in: `段一

			段二`,
			want: "段一\n\n段二",
		},
		{
			name: "单行：原样（无第二行可对齐）",
			in:   "只有一行",
			want: "只有一行",
		},
		{
			name: "空字符串",
			in:   "",
			want: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Dedent(c.in)
			if got != c.want {
				t.Fatalf("Dedent(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

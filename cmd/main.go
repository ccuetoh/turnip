package main

import (
	"fmt"

	"turnip"
)

func main() {
	type s1 struct {
		Foo     string
		Shared  int
		private func() bool
	}

	type s2 struct {
		Bar    string
		Shared int
	}

	u, err := turnip.New(
		turnip.Candidate(s1{}),
		turnip.Candidate(s2{}),
		turnip.EnableDebug(),
	)
	if err != nil {
		panic(err)
	}

	res, err := u.UnmarshalJSON([]byte(`
		{
			"foo": "s1",
			"shared": 1
		}
	`))
	if err != nil {
		panic(err)
	}

	res = res.(*s1)
	fmt.Printf("%+v\n", res)
}

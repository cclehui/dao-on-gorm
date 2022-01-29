package ctime

import (
	"context"
	"fmt"
	"time"
)

func ExampleDuration_UnmarshalText() {
	var d Duration
	err := d.UnmarshalText([]byte("1s"))
	if err != nil {
		return
	}
	fmt.Printf("%v\n", time.Duration(d) == time.Second)

	// Output: true
}

func ExampleDuration_Shrink() {
	d := Duration(time.Second * 5)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	d, ctx, cancel = d.Shrink(ctx)
	defer cancel()
	fmt.Printf("%v\n", time.Duration(d) == time.Second*5)

	d = Duration(time.Second * 5)
	d, ctx, cancel = d.Shrink(context.Background())
	defer cancel()
	_, ok := ctx.Deadline()
	fmt.Printf("%v\n", ok)

	// Output:
	// false
	// true
}

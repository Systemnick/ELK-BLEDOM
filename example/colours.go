package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.fergus.london/bledom/device"
)

var commands []*device.ColourCommand = []*device.ColourCommand{
	device.ColourCommandFromValues(0x80, 0x00, 0x00),
	device.ColourCommandFromValues(0x80, 0x80, 0x00),
	device.ColourCommandFromValues(0x80, 0x00, 0x80),
	device.ColourCommandFromValues(0x80, 0x80, 0x80),
}

func gracefulTermination(cancel context.CancelFunc, d *device.BleDom) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("SIGTERM received, performing graceful shutdown")
		cancel()
		d.Stop()
	}()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	d := device.NewBleDomRGBController(ctx, device.Options{})

	if err := d.Connect(time.Duration(time.Minute)); err != nil {
		panic(err)
	}

	gracefulTermination(cancel, d)

	// Don't terminate until device has been cleanly shutdown
	defer func() {
		fmt.Println("waiting for device disconnection")
		d.Done()
		fmt.Println("device disconnected: terminating app")
	}()

	d.PollState(time.Duration(30)*time.Second, func(state []byte) {
		fmt.Println("recieved latest state from device", state)
	})

	fmt.Println("setting brightness to maximum")
	d.WriteCommand(device.BrightnessCommandFromValue(0x80))

	i := 1
	for {
		t := time.NewTimer(time.Second)

		select {
		case <-ctx.Done():
			fmt.Println("context cancelled: stopping commands")
			t.Stop()
			return
		case <-t.C:
			d.WriteCommand(commands[len(commands)%i])

			i = i + 1
			t.Reset(time.Second)
		}
	}
}

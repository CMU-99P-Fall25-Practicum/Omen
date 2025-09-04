package main

import (
	"net/netip"
	"os"
	"time"

	"github.com/rs/zerolog"
)

func main() {
	// slurp input
	var sshAddr netip.AddrPort
	// check for output redirection

	// spawn a logger (assumes STDOUT)
	log := zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}).With().Timestamp().Logger()

	// dial SSH (if specified in input)
	if sshAddr.IsValid() {
		log.Info().Str("target ip", sshAddr.Addr().String()).Func(func(e *zerolog.Event) {
			if sshAddr.Port() != 22 {
				e.Uint16("non-standard port", sshAddr.Port())
			}
		}).Msg("dialing ssh...")
	}

	// profit

}

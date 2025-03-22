package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"

	"github.com/yl2chen/cidranger"

	"github.com/TecharoHQ/anubis/lib/policy/config"
)

func ParseConfig(fin io.Reader, fname string, defaultDifficulty int) (*ParsedConfig, error) {
	var c config.Config
	if err := json.NewDecoder(fin).Decode(&c); err != nil {
		return nil, fmt.Errorf("can't parse policy config JSON %s: %w", fname, err)
	}

	if err := c.Valid(); err != nil {
		return nil, err
	}

	var err error

	result := NewParsedConfig(c)

	for _, b := range c.Bots {
		if berr := b.Valid(); berr != nil {
			err = errors.Join(err, berr)
			continue
		}

		var botParseErr error
		parsedBot := Bot{
			Name:   b.Name,
			Action: b.Action,
		}

		if b.RemoteAddr != nil && len(b.RemoteAddr) > 0 {
			parsedBot.Ranger = cidranger.NewPCTrieRanger()

			for _, cidr := range b.RemoteAddr {
				_, rng, err := net.ParseCIDR(cidr)
				if err != nil {
					return nil, fmt.Errorf("[unexpected] range %s not parsing: %w", cidr, err)
				}

				parsedBot.Ranger.Insert(cidranger.NewBasicRangerEntry(*rng))
			}
		}

		if b.UserAgentRegex != nil {
			userAgent, err := regexp.Compile(*b.UserAgentRegex)
			if err != nil {
				botParseErr = errors.Join(botParseErr, fmt.Errorf("while compiling user agent regexp: %w", err))
				continue
			} else {
				parsedBot.UserAgent = userAgent
			}
		}

		if b.PathRegex != nil {
			path, err := regexp.Compile(*b.PathRegex)
			if err != nil {
				botParseErr = errors.Join(botParseErr, fmt.Errorf("while compiling path regexp: %w", err))
				continue
			} else {
				parsedBot.Path = path
			}
		}

		if b.Challenge == nil {
			parsedBot.Challenge = &config.ChallengeRules{
				Difficulty: defaultDifficulty,
				ReportAs:   defaultDifficulty,
				Algorithm:  config.AlgorithmFast,
			}
		} else {
			parsedBot.Challenge = b.Challenge
			if parsedBot.Challenge.Algorithm == config.AlgorithmUnknown {
				parsedBot.Challenge.Algorithm = config.AlgorithmFast
			}
		}

		result.Bots = append(result.Bots, parsedBot)
	}

	if err != nil {
		return nil, fmt.Errorf("errors validating policy config JSON %s: %w", fname, err)
	}

	result.DNSBL = c.DNSBL

	return result, nil
}

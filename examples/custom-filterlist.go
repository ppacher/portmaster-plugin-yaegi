package main

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/safing/portmaster/plugin/shared/proto"
)

var (
	readListOnce  sync.Once
	domainList    map[string]struct{}
	readListError error
)

func DecideOnConnection(ctx context.Context, conn *proto.Connection) (proto.Verdict, string, error) {
	readListOnce.Do(func() {
		domainList = make(map[string]struct{})

		var content []byte
		content, readListError = os.ReadFile("/tmp/custom-filter-list")

		for _, line := range strings.Split(string(content), "\n") {
			if !strings.HasSuffix(line, ".") {
				line += "."
			}
			domainList[line] = struct{}{}
		}

		log.Printf("[INFO] loaded %d domain entries from /tmp/custom-filter-list", len(domainList))
	})
	if readListError != nil {
		log.Printf("[FAIl] failed to load custom domain list: %s", readListError)

		return proto.Verdict_VERDICT_FAILED, "", readListError
	}

	log.Printf("evaluating %s against custom filter list", conn.GetId())

	domain := conn.GetEntity().GetDomain()
	if domain == "" {
		return proto.Verdict_VERDICT_UNDETERMINABLE, "", nil
	}

	parts := strings.Split(domain, ".")

	for idx := range parts {
		// Don't check the top-level domain (i.e. .com., .at., ...)
		if idx == len(parts)-2 {
			break
		}

		d := strings.Join(parts[idx:], ".")
		log.Printf("checking against %q", d)

		if _, ok := domainList[d]; ok {
			return proto.Verdict_VERDICT_BLOCK, d + " is blocked", nil
		}
	}

	return proto.Verdict_VERDICT_UNDECIDED, "", readListError
}

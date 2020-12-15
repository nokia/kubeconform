package resource

import (
	"bufio"
	"bytes"
	"context"
	"io"
)

// SplitYAMLDocument is a bufio.SplitFunc for splitting a YAML document into individual documents.
//
// This is from Kubernetes' 'pkg/util/yaml'.splitYAMLDocument, which is unfortunately
// not exported.
func SplitYAMLDocument(data []byte, atEOF bool) (advance int, token []byte, err error) {
	const yamlSeparator = "\n---"
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	sep := len([]byte(yamlSeparator))
	if i := bytes.Index(data, []byte(yamlSeparator)); i >= 0 {
		// We have a potential document terminator
		i += sep
		after := data[i:]
		if len(after) == 0 {
			// we can't read any more characters
			if atEOF {
				return len(data), data[:len(data)-sep], nil
			}
			return 0, nil, nil
		}
		if j := bytes.IndexByte(after, '\n'); j >= 0 {
			return i + j + 1, data[0 : i-sep], nil
		}
		return 0, nil, nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

// FromStream reads resources from a byte stream, usually here stdin
func FromStream(ctx context.Context, path string, r io.Reader) (<-chan Resource, <-chan error) {
	resources := make(chan Resource)
	errors := make(chan error)

	go func() {
		scanner := bufio.NewScanner(r)
		const maxResourceSize = 4 * 1024 * 1024 // 4MB ought to be enough for everybody
		buf := make([]byte, maxResourceSize)
		scanner.Buffer(buf, maxResourceSize)
		scanner.Split(SplitYAMLDocument)

	SCAN:
		for res := scanner.Scan(); res != false; res = scanner.Scan() {
			select {
			case <-ctx.Done():
				break SCAN
			default:
			}
			resources <- Resource{Path: path, Bytes: scanner.Bytes()}
		}

		close(resources)
		close(errors)
	}()

	return resources, errors
}

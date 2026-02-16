package channel_test

import (
	"context"
	"testing"

	"github.com/Kxiandaoyan/Memoh-v2/internal/channel"
)

const dirTestChannelType = channel.ChannelType("dir-test")

// dirMockAdapter implements Adapter and ChannelDirectoryAdapter for registry DirectoryAdapter tests.
type dirMockAdapter struct{}

func (a *dirMockAdapter) Type() channel.ChannelType { return dirTestChannelType }

func (a *dirMockAdapter) Descriptor() channel.Descriptor {
	return channel.Descriptor{Type: dirTestChannelType, DisplayName: "DirTest"}
}

func (a *dirMockAdapter) ListPeers(ctx context.Context, cfg channel.ChannelConfig, query channel.DirectoryQuery) ([]channel.DirectoryEntry, error) {
	return nil, nil
}

func (a *dirMockAdapter) ListGroups(ctx context.Context, cfg channel.ChannelConfig, query channel.DirectoryQuery) ([]channel.DirectoryEntry, error) {
	return nil, nil
}

func (a *dirMockAdapter) ListGroupMembers(ctx context.Context, cfg channel.ChannelConfig, groupID string, query channel.DirectoryQuery) ([]channel.DirectoryEntry, error) {
	return nil, nil
}

func (a *dirMockAdapter) ResolveEntry(ctx context.Context, cfg channel.ChannelConfig, input string, kind channel.DirectoryEntryKind) (channel.DirectoryEntry, error) {
	return channel.DirectoryEntry{}, nil
}

func TestDirectoryAdapter_Unsupported(t *testing.T) {
	t.Parallel()
	reg := newTestConfigRegistry()
	dir, ok := reg.DirectoryAdapter(testChannelType)
	if ok || dir != nil {
		t.Fatalf("DirectoryAdapter(test) = (%v, %v), want (nil, false)", dir, ok)
	}
}

func TestDirectoryAdapter_Supported(t *testing.T) {
	t.Parallel()
	reg := channel.NewRegistry()
	reg.MustRegister(&dirMockAdapter{})
	dir, ok := reg.DirectoryAdapter(dirTestChannelType)
	if !ok || dir == nil {
		t.Fatalf("DirectoryAdapter(dir-test) = (%v, %v), want (non-nil, true)", dir, ok)
	}
}

func TestDirectoryAdapter_UnknownType(t *testing.T) {
	t.Parallel()
	reg := channel.NewRegistry()
	dir, ok := reg.DirectoryAdapter(channel.ChannelType("unknown"))
	if ok || dir != nil {
		t.Fatalf("DirectoryAdapter(unknown) = (%v, %v), want (nil, false)", dir, ok)
	}
}

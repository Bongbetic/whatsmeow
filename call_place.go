// Copyright (c) 2026 Bongbetic contributors (WL-5.1 fork delta).
//
// This file is the Bongbetic/whatsmeow fork addition on top of
// upstream tulir/whatsmeow: outbound call signaling (offer,
// preaccept, accept, terminate). Framing mirrors upstream
// RejectCall and the inbound shapes in handleCallEvent; the media
// contract (relay endpoints, keys) arrives on CallTransport events
// once the peer answers. Verified live, not headless: place a call
// and watch the peer ring; captures in tools/interop/phase5.md.

package whatsmeow

import (
	"context"

	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

// callWrapper frames one call-signaling child the way RejectCall does:
// <call id from=own to=peer><child/></call> with non-AD JIDs.
func (cli *Client) callWrapper(to types.JID, child waBinary.Node) (waBinary.Node, error) {
	ownID := cli.getOwnID()
	if ownID.IsEmpty() {
		return waBinary.Node{}, ErrNotLoggedIn
	}
	ownID, to = ownID.ToNonAD(), to.ToNonAD()
	return waBinary.Node{
		Tag:   "call",
		Attrs: waBinary.Attrs{"id": cli.GenerateMessageID(), "from": ownID, "to": to},
		Content: []waBinary.Node{
			child,
		},
	}, nil
}

// PlaceCall sends a 1:1 call offer and returns the call ID the peer
// will echo on preaccept/accept/terminate. video selects the media
// marker. The peer ringing (then preaccept → accept → transport
// events here) is the live proof; absence with no error means the
// server dropped the framing — capture and iterate, do not retry
// blindly in a loop.
func (cli *Client) PlaceCall(ctx context.Context, to types.JID, video bool) (string, error) {
	ownID := cli.getOwnID()
	if ownID.IsEmpty() {
		return "", ErrNotLoggedIn
	}
	callID := string(cli.GenerateMessageID())
	media := "audio"
	if video {
		media = "video"
	}
	offer := waBinary.Node{
		Tag: "offer",
		Attrs: waBinary.Attrs{
			"call-id":      callID,
			"call-creator": ownID.ToNonAD(),
			"media":        media,
		},
	}
	wrapper, err := cli.callWrapper(to, offer)
	if err != nil {
		return "", err
	}
	if err := cli.sendNode(ctx, wrapper); err != nil {
		return "", err
	}
	return callID, nil
}

// PreAcceptCall acknowledges an incoming offer before media flows
// (ringing → preaccept). from is the caller, callID the offered ID.
func (cli *Client) PreAcceptCall(ctx context.Context, from types.JID, callID string) error {
	ownID := cli.getOwnID()
	if ownID.IsEmpty() {
		return ErrNotLoggedIn
	}
	pre := waBinary.Node{
		Tag:   "preaccept",
		Attrs: waBinary.Attrs{"call-id": callID, "call-creator": from.ToNonAD()},
	}
	wrapper, err := cli.callWrapper(from, pre)
	if err != nil {
		return err
	}
	return cli.sendNode(ctx, wrapper)
}

// AcceptCall accepts an incoming call with media. from is the caller
// (or call-creator for group legs), callID the offered ID.
func (cli *Client) AcceptCall(ctx context.Context, from types.JID, callID string, video bool) error {
	ownID := cli.getOwnID()
	if ownID.IsEmpty() {
		return ErrNotLoggedIn
	}
	media := "audio"
	if video {
		media = "video"
	}
	acc := waBinary.Node{
		Tag: "accept",
		Attrs: waBinary.Attrs{
			"call-id":      callID,
			"call-creator": from.ToNonAD(),
			"media":        media,
		},
	}
	wrapper, err := cli.callWrapper(from, acc)
	if err != nil {
		return err
	}
	return cli.sendNode(ctx, wrapper)
}

// TerminateCall ends an answered leg. Unanswered legs should use
// RejectCall (upstream, proven); terminate is for live media legs.
// reason is passed through verbatim (e.g. "user-hangup", "timeout").
func (cli *Client) TerminateCall(ctx context.Context, peer types.JID, callID, reason string) error {
	ownID := cli.getOwnID()
	if ownID.IsEmpty() {
		return ErrNotLoggedIn
	}
	term := waBinary.Node{
		Tag: "terminate",
		Attrs: waBinary.Attrs{
			"call-id":      callID,
			"call-creator": ownID.ToNonAD(),
			"reason":       reason,
		},
	}
	wrapper, err := cli.callWrapper(peer, term)
	if err != nil {
		return err
	}
	return cli.sendNode(ctx, wrapper)
}

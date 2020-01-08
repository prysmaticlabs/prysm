package connmgr

import (
	"context"
	"testing"
	"time"

	detectrace "github.com/ipfs/go-detect-race"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"

	tu "github.com/libp2p/go-libp2p-core/test"
	ma "github.com/multiformats/go-multiaddr"
)

type tconn struct {
	network.Conn

	peer             peer.ID
	closed           bool
	disconnectNotify func(net network.Network, conn network.Conn)
}

func (c *tconn) Close() error {
	c.closed = true
	if c.disconnectNotify != nil {
		c.disconnectNotify(nil, c)
	}
	return nil
}

func (c *tconn) RemotePeer() peer.ID {
	return c.peer
}

func (c *tconn) RemoteMultiaddr() ma.Multiaddr {
	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/udp/1234")
	if err != nil {
		panic("cannot create multiaddr")
	}
	return addr
}

func randConn(t testing.TB, discNotify func(network.Network, network.Conn)) network.Conn {
	pid := tu.RandPeerIDFatal(t)
	return &tconn{peer: pid, disconnectNotify: discNotify}
}

func TestConnTrimming(t *testing.T) {
	cm := NewConnManager(200, 300, 0)
	not := cm.Notifee()

	var conns []network.Conn
	for i := 0; i < 300; i++ {
		rc := randConn(t, nil)
		conns = append(conns, rc)
		not.Connected(nil, rc)
	}

	for _, c := range conns {
		if c.(*tconn).closed {
			t.Fatal("nothing should be closed yet")
		}
	}

	for i := 0; i < 100; i++ {
		cm.TagPeer(conns[i].RemotePeer(), "foo", 10)
	}

	cm.TagPeer(conns[299].RemotePeer(), "badfoo", -5)

	cm.TrimOpenConns(context.Background())

	for i := 0; i < 100; i++ {
		c := conns[i]
		if c.(*tconn).closed {
			t.Fatal("these shouldnt be closed")
		}
	}

	if !conns[299].(*tconn).closed {
		t.Fatal("conn with bad tag should have gotten closed")
	}
}

func TestConnsToClose(t *testing.T) {
	cm := NewConnManager(0, 10, 0)
	conns := cm.getConnsToClose(context.Background())
	if conns != nil {
		t.Fatal("expected no connections")
	}

	cm = NewConnManager(10, 0, 0)
	conns = cm.getConnsToClose(context.Background())
	if conns != nil {
		t.Fatal("expected no connections")
	}

	cm = NewConnManager(1, 1, 0)
	conns = cm.getConnsToClose(context.Background())
	if conns != nil {
		t.Fatal("expected no connections")
	}

	cm = NewConnManager(1, 1, 10*time.Minute)
	not := cm.Notifee()
	for i := 0; i < 5; i++ {
		conn := randConn(t, nil)
		not.Connected(nil, conn)
	}
	conns = cm.getConnsToClose(context.Background())
	if len(conns) != 0 {
		t.Fatal("expected no connections")
	}
}

func TestGetTagInfo(t *testing.T) {
	start := time.Now()
	cm := NewConnManager(1, 1, 10*time.Minute)
	not := cm.Notifee()
	conn := randConn(t, nil)
	not.Connected(nil, conn)
	end := time.Now()

	other := tu.RandPeerIDFatal(t)
	tag := cm.GetTagInfo(other)
	if tag != nil {
		t.Fatal("expected no tag")
	}

	tag = cm.GetTagInfo(conn.RemotePeer())
	if tag == nil {
		t.Fatal("expected tag")
	}
	if tag.FirstSeen.Before(start) || tag.FirstSeen.After(end) {
		t.Fatal("expected first seen time")
	}
	if tag.Value != 0 {
		t.Fatal("expected zero value")
	}
	if len(tag.Tags) != 0 {
		t.Fatal("expected no tags")
	}
	if len(tag.Conns) != 1 {
		t.Fatal("expected one connection")
	}
	for s, tm := range tag.Conns {
		if s != conn.RemoteMultiaddr().String() {
			t.Fatal("unexpected multiaddr")
		}
		if tm.Before(start) || tm.After(end) {
			t.Fatal("unexpected connection time")
		}
	}

	cm.TagPeer(conn.RemotePeer(), "tag", 5)
	tag = cm.GetTagInfo(conn.RemotePeer())
	if tag == nil {
		t.Fatal("expected tag")
	}
	if tag.FirstSeen.Before(start) || tag.FirstSeen.After(end) {
		t.Fatal("expected first seen time")
	}
	if tag.Value != 5 {
		t.Fatal("expected five value")
	}
	if len(tag.Tags) != 1 {
		t.Fatal("expected no tags")
	}
	for tString, v := range tag.Tags {
		if tString != "tag" || v != 5 {
			t.Fatal("expected tag value")
		}
	}
	if len(tag.Conns) != 1 {
		t.Fatal("expected one connection")
	}
	for s, tm := range tag.Conns {
		if s != conn.RemoteMultiaddr().String() {
			t.Fatal("unexpected multiaddr")
		}
		if tm.Before(start) || tm.After(end) {
			t.Fatal("unexpected connection time")
		}
	}
}

func TestTagPeerNonExistant(t *testing.T) {
	cm := NewConnManager(1, 1, 10*time.Minute)

	id := tu.RandPeerIDFatal(t)
	cm.TagPeer(id, "test", 1)

	if !cm.segments.get(id).peers[id].temp {
		t.Fatal("expected 1 temporary entry")
	}
}

func TestUntagPeer(t *testing.T) {
	cm := NewConnManager(1, 1, 10*time.Minute)
	not := cm.Notifee()
	conn := randConn(t, nil)
	not.Connected(nil, conn)
	rp := conn.RemotePeer()
	cm.TagPeer(rp, "tag", 5)
	cm.TagPeer(rp, "tag two", 5)

	id := tu.RandPeerIDFatal(t)
	cm.UntagPeer(id, "test")
	if len(cm.segments.get(rp).peers[rp].tags) != 2 {
		t.Fatal("expected tags to be uneffected")
	}

	cm.UntagPeer(conn.RemotePeer(), "test")
	if len(cm.segments.get(rp).peers[rp].tags) != 2 {
		t.Fatal("expected tags to be uneffected")
	}

	cm.UntagPeer(conn.RemotePeer(), "tag")
	if len(cm.segments.get(rp).peers[rp].tags) != 1 {
		t.Fatal("expected tag to be removed")
	}
	if cm.segments.get(rp).peers[rp].value != 5 {
		t.Fatal("expected aggreagte tag value to be 5")
	}
}

func TestGetInfo(t *testing.T) {
	start := time.Now()
	gp := 10 * time.Minute
	cm := NewConnManager(1, 5, gp)
	not := cm.Notifee()
	conn := randConn(t, nil)
	not.Connected(nil, conn)
	cm.TrimOpenConns(context.Background())
	end := time.Now()

	info := cm.GetInfo()
	if info.HighWater != 5 {
		t.Fatal("expected highwater to be 5")
	}
	if info.LowWater != 1 {
		t.Fatal("expected highwater to be 1")
	}
	if info.LastTrim.Before(start) || info.LastTrim.After(end) {
		t.Fatal("unexpected last trim time")
	}
	if info.GracePeriod != gp {
		t.Fatal("unexpected grace period")
	}
	if info.ConnCount != 1 {
		t.Fatal("unexpected number of connections")
	}
}

func TestDoubleConnection(t *testing.T) {
	gp := 10 * time.Minute
	cm := NewConnManager(1, 5, gp)
	not := cm.Notifee()
	conn := randConn(t, nil)
	not.Connected(nil, conn)
	cm.TagPeer(conn.RemotePeer(), "foo", 10)
	not.Connected(nil, conn)
	if cm.connCount != 1 {
		t.Fatal("unexpected number of connections")
	}
	if cm.segments.get(conn.RemotePeer()).peers[conn.RemotePeer()].value != 10 {
		t.Fatal("unexpected peer value")
	}
}

func TestDisconnected(t *testing.T) {
	gp := 10 * time.Minute
	cm := NewConnManager(1, 5, gp)
	not := cm.Notifee()
	conn := randConn(t, nil)
	not.Connected(nil, conn)
	cm.TagPeer(conn.RemotePeer(), "foo", 10)

	not.Disconnected(nil, randConn(t, nil))
	if cm.connCount != 1 {
		t.Fatal("unexpected number of connections")
	}
	if cm.segments.get(conn.RemotePeer()).peers[conn.RemotePeer()].value != 10 {
		t.Fatal("unexpected peer value")
	}

	not.Disconnected(nil, &tconn{peer: conn.RemotePeer()})
	if cm.connCount != 1 {
		t.Fatal("unexpected number of connections")
	}
	if cm.segments.get(conn.RemotePeer()).peers[conn.RemotePeer()].value != 10 {
		t.Fatal("unexpected peer value")
	}

	not.Disconnected(nil, conn)
	if cm.connCount != 0 {
		t.Fatal("unexpected number of connections")
	}
	if cm.segments.countPeers() != 0 {
		t.Fatal("unexpected number of peers")
	}
}

func TestGracePeriod(t *testing.T) {
	if detectrace.WithRace() {
		t.Skip("race detector is unhappy with this test")
	}

	SilencePeriod = 0
	cm := NewConnManager(10, 20, 100*time.Millisecond)
	SilencePeriod = 10 * time.Second

	not := cm.Notifee()

	var conns []network.Conn

	// Add a connection and wait the grace period.
	{
		rc := randConn(t, not.Disconnected)
		conns = append(conns, rc)
		not.Connected(nil, rc)

		time.Sleep(200 * time.Millisecond)

		if rc.(*tconn).closed {
			t.Fatal("expected conn to remain open")
		}
	}

	// quickly add 30 connections (sending us above the high watermark)
	for i := 0; i < 30; i++ {
		rc := randConn(t, not.Disconnected)
		conns = append(conns, rc)
		not.Connected(nil, rc)
	}

	cm.TrimOpenConns(context.Background())

	for _, c := range conns {
		if c.(*tconn).closed {
			t.Fatal("expected no conns to be closed")
		}
	}

	time.Sleep(200 * time.Millisecond)

	cm.TrimOpenConns(context.Background())

	closed := 0
	for _, c := range conns {
		if c.(*tconn).closed {
			closed++
		}
	}

	if closed != 21 {
		t.Fatal("expected to have closed 21 connections")
	}
}

// see https://github.com/libp2p/go-libp2p-connmgr/issues/23
func TestQuickBurstRespectsSilencePeriod(t *testing.T) {
	if detectrace.WithRace() {
		t.Skip("race detector is unhappy with this test")
	}

	cm := NewConnManager(10, 20, 0)
	not := cm.Notifee()

	var conns []network.Conn

	// quickly produce 30 connections (sending us above the high watermark)
	for i := 0; i < 30; i++ {
		rc := randConn(t, not.Disconnected)
		conns = append(conns, rc)
		not.Connected(nil, rc)
	}

	// wait for a few seconds
	time.Sleep(time.Second * 3)

	// only the first trim is allowed in; make sure we close at most 20 connections, not all of them.
	var closed int
	for _, c := range conns {
		if c.(*tconn).closed {
			closed++
		}
	}
	if closed > 20 {
		t.Fatalf("should have closed at most 20 connections, closed: %d", closed)
	}
	if total := closed + int(cm.connCount); total != 30 {
		t.Fatalf("expected closed connections + open conn count to equal 30, value: %d", total)
	}
}

func TestPeerProtectionSingleTag(t *testing.T) {
	if detectrace.WithRace() {
		t.Skip("race detector is unhappy with this test")
	}

	SilencePeriod = 0
	cm := NewConnManager(19, 20, 0)
	SilencePeriod = 10 * time.Second

	not := cm.Notifee()

	var conns []network.Conn
	addConn := func(value int) {
		rc := randConn(t, not.Disconnected)
		conns = append(conns, rc)
		not.Connected(nil, rc)
		cm.TagPeer(rc.RemotePeer(), "test", value)
	}

	// produce 20 connections with unique peers.
	for i := 0; i < 20; i++ {
		addConn(20)
	}

	// protect the first 5 peers.
	var protected []network.Conn
	for _, c := range conns[0:5] {
		cm.Protect(c.RemotePeer(), "global")
		protected = append(protected, c)
		// tag them negatively to make them preferred for pruning.
		cm.TagPeer(c.RemotePeer(), "test", -100)
	}

	// add 1 more conn, this shouldn't send us over the limit as protected conns don't count
	addConn(20)

	cm.TrimOpenConns(context.Background())

	for _, c := range conns {
		if c.(*tconn).closed {
			t.Error("connection was closed by connection manager")
		}
	}

	// add 5 more connection, sending the connection manager overboard.
	for i := 0; i < 5; i++ {
		addConn(20)
	}

	cm.TrimOpenConns(context.Background())

	for _, c := range protected {
		if c.(*tconn).closed {
			t.Error("protected connection was closed by connection manager")
		}
	}

	closed := 0
	for _, c := range conns {
		if c.(*tconn).closed {
			closed++
		}
	}
	if closed != 2 {
		t.Errorf("expected 2 connection to be closed, found %d", closed)
	}

	// unprotect the first peer.
	cm.Unprotect(protected[0].RemotePeer(), "global")

	// add 2 more connections, sending the connection manager overboard again.
	for i := 0; i < 2; i++ {
		addConn(20)
	}

	cm.TrimOpenConns(context.Background())

	if !protected[0].(*tconn).closed {
		t.Error("unprotected connection was kept open by connection manager")
	}
	for _, c := range protected[1:] {
		if c.(*tconn).closed {
			t.Error("protected connection was closed by connection manager")
		}
	}
}

func TestPeerProtectionMultipleTags(t *testing.T) {
	if detectrace.WithRace() {
		t.Skip("race detector is unhappy with this test")
	}

	SilencePeriod = 0
	cm := NewConnManager(19, 20, 0)
	SilencePeriod = 10 * time.Second

	not := cm.Notifee()

	// produce 20 connections with unique peers.
	var conns []network.Conn
	for i := 0; i < 20; i++ {
		rc := randConn(t, not.Disconnected)
		conns = append(conns, rc)
		not.Connected(nil, rc)
		cm.TagPeer(rc.RemotePeer(), "test", 20)
	}

	// protect the first 5 peers under two tags.
	var protected []network.Conn
	for _, c := range conns[0:5] {
		cm.Protect(c.RemotePeer(), "tag1")
		cm.Protect(c.RemotePeer(), "tag2")
		protected = append(protected, c)
		// tag them negatively to make them preferred for pruning.
		cm.TagPeer(c.RemotePeer(), "test", -100)
	}

	// add one more connection, sending the connection manager overboard.
	not.Connected(nil, randConn(t, not.Disconnected))

	cm.TrimOpenConns(context.Background())

	for _, c := range protected {
		if c.(*tconn).closed {
			t.Error("protected connection was closed by connection manager")
		}
	}

	// remove the protection from one tag.
	for _, c := range protected {
		if !cm.Unprotect(c.RemotePeer(), "tag1") {
			t.Error("peer should still be protected")
		}
	}

	// add 2 more connections, sending the connection manager overboard again.
	for i := 0; i < 2; i++ {
		rc := randConn(t, not.Disconnected)
		conns = append(conns, rc)
		not.Connected(nil, rc)
		cm.TagPeer(rc.RemotePeer(), "test", 20)
	}

	cm.TrimOpenConns(context.Background())

	// connections should still remain open, as they were protected.
	for _, c := range protected[0:] {
		if c.(*tconn).closed {
			t.Error("protected connection was closed by connection manager")
		}
	}

	// unprotect the first peer entirely.
	cm.Unprotect(protected[0].RemotePeer(), "tag2")

	// add 2 more connections, sending the connection manager overboard again.
	for i := 0; i < 2; i++ {
		rc := randConn(t, not.Disconnected)
		conns = append(conns, rc)
		not.Connected(nil, rc)
		cm.TagPeer(rc.RemotePeer(), "test", 20)
	}

	cm.TrimOpenConns(context.Background())

	if !protected[0].(*tconn).closed {
		t.Error("unprotected connection was kept open by connection manager")
	}
	for _, c := range protected[1:] {
		if c.(*tconn).closed {
			t.Error("protected connection was closed by connection manager")
		}
	}

}

func TestPeerProtectionIdempotent(t *testing.T) {
	SilencePeriod = 0
	cm := NewConnManager(10, 20, 0)
	SilencePeriod = 10 * time.Second

	id, _ := tu.RandPeerID()
	cm.Protect(id, "global")
	cm.Protect(id, "global")
	cm.Protect(id, "global")
	cm.Protect(id, "global")

	if len(cm.protected[id]) > 1 {
		t.Error("expected peer to be protected only once")
	}

	if !cm.Unprotect(id, "unused") {
		t.Error("expected peer to continue to be protected")
	}

	if !cm.Unprotect(id, "unused2") {
		t.Error("expected peer to continue to be protected")
	}

	if cm.Unprotect(id, "global") {
		t.Error("expected peer to be unprotected")
	}

	if len(cm.protected) > 0 {
		t.Error("expected no protections")
	}
}

func TestUpsertTag(t *testing.T) {
	cm := NewConnManager(1, 1, 10*time.Minute)
	not := cm.Notifee()
	conn := randConn(t, nil)
	rp := conn.RemotePeer()

	// this is an early tag, before the Connected notification arrived.
	cm.UpsertTag(rp, "tag", func(v int) int { return v + 1 })
	if len(cm.segments.get(rp).peers[rp].tags) != 1 {
		t.Fatal("expected a tag")
	}
	if cm.segments.get(rp).peers[rp].value != 1 {
		t.Fatal("expected a tag value of 1")
	}

	// now let's notify the connection.
	not.Connected(nil, conn)

	cm.UpsertTag(rp, "tag", func(v int) int { return v + 1 })
	if len(cm.segments.get(rp).peers[rp].tags) != 1 {
		t.Fatal("expected a tag")
	}
	if cm.segments.get(rp).peers[rp].value != 2 {
		t.Fatal("expected a tag value of 2")
	}

	cm.UpsertTag(rp, "tag", func(v int) int { return v - 1 })
	if len(cm.segments.get(rp).peers[rp].tags) != 1 {
		t.Fatal("expected a tag")
	}
	if cm.segments.get(rp).peers[rp].value != 1 {
		t.Fatal("expected a tag value of 1")
	}
}

func TestTemporaryEntriesClearedFirst(t *testing.T) {
	cm := NewConnManager(1, 1, 0)

	id := tu.RandPeerIDFatal(t)
	cm.TagPeer(id, "test", 20)

	if cm.GetTagInfo(id).Value != 20 {
		t.Fatal("expected an early tag with value 20")
	}

	not := cm.Notifee()
	conn1, conn2 := randConn(t, nil), randConn(t, nil)
	not.Connected(nil, conn1)
	not.Connected(nil, conn2)

	cm.TrimOpenConns(context.Background())
	if cm.GetTagInfo(id) != nil {
		t.Fatal("expected no temporary tags after trimming")
	}
}

func TestTemporaryEntryConvertedOnConnection(t *testing.T) {
	cm := NewConnManager(1, 1, 0)

	conn := randConn(t, nil)
	cm.TagPeer(conn.RemotePeer(), "test", 20)

	ti := cm.segments.get(conn.RemotePeer()).peers[conn.RemotePeer()]

	if ti.value != 20 || !ti.temp {
		t.Fatal("expected a temporary tag with value 20")
	}

	not := cm.Notifee()
	not.Connected(nil, conn)

	if ti.value != 20 || ti.temp {
		t.Fatal("expected a non-temporary tag with value 20")
	}
}

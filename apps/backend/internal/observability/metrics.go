package observability

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var tickBuckets = [...]float64{0.001, 0.005, 0.010, 0.025, 0.050, 0.100, 0.250}

type TickSample struct {
	Room           string
	Total          time.Duration
	Movement       time.Duration
	Weapons        time.Duration
	ProjectileMove time.Duration
	BroadPhase     time.Duration
	NarrowPhase    time.Duration
	EnemyAI        time.Duration
	Pickups        time.Duration
	Spawning       time.Duration
	Players        int
	Monsters       int
	Projectiles    int
	PickupsCount   int
	CandidatePairs uint64
	NarrowChecks   uint64
	ConfirmedHits  uint64
}

type roomGauge struct {
	players     int
	monsters    int
	projectiles int
	pickups     int
	queueDepth  int
	queueMax    int
}

type durationAggregate struct {
	count uint64
	sum   time.Duration
}

type Collector struct {
	mu                    sync.Mutex
	rooms                 map[string]roomGauge
	tickCount             uint64
	tickSum               time.Duration
	tickBuckets           [len(tickBuckets) + 1]uint64
	phases                map[string]durationAggregate
	candidatePairs        uint64
	narrowChecks          uint64
	confirmedHits         uint64
	snapshotBuild         durationAggregate
	snapshotEncode        durationAggregate
	snapshotBytes         uint64
	websocketBytes        uint64
	websocketEnqueue      durationAggregate
	snapshotReplaced      uint64
	criticalQueueFailures uint64
}

func NewCollector() *Collector {
	return &Collector{
		rooms:  make(map[string]roomGauge),
		phases: make(map[string]durationAggregate),
	}
}

func (c *Collector) RecordTick(sample TickSample) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tickCount++
	c.tickSum += sample.Total
	placed := false
	for index, upper := range tickBuckets {
		if sample.Total.Seconds() <= upper {
			c.tickBuckets[index]++
			placed = true
			break
		}
	}
	if !placed {
		c.tickBuckets[len(tickBuckets)]++
	}
	for name, duration := range map[string]time.Duration{
		"movement": sample.Movement, "weapons": sample.Weapons, "projectile_move": sample.ProjectileMove,
		"broad_phase": sample.BroadPhase, "narrow_phase": sample.NarrowPhase, "enemy_ai": sample.EnemyAI,
		"pickups": sample.Pickups, "spawning": sample.Spawning,
	} {
		aggregate := c.phases[name]
		aggregate.count++
		aggregate.sum += duration
		c.phases[name] = aggregate
	}
	c.candidatePairs += sample.CandidatePairs
	c.narrowChecks += sample.NarrowChecks
	c.confirmedHits += sample.ConfirmedHits
	gauge := c.rooms[sample.Room]
	gauge.players = sample.Players
	gauge.monsters = sample.Monsters
	gauge.projectiles = sample.Projectiles
	gauge.pickups = sample.PickupsCount
	c.rooms[sample.Room] = gauge
}

func (c *Collector) RecordSnapshotBuild(duration time.Duration) {
	c.mu.Lock()
	c.snapshotBuild.count++
	c.snapshotBuild.sum += duration
	c.mu.Unlock()
}

func (c *Collector) RecordMessageEncode(duration time.Duration, bytes int, snapshot bool) {
	c.mu.Lock()
	encodedBytes := uint64(max(0, bytes))
	c.websocketBytes += encodedBytes
	if snapshot {
		c.snapshotEncode.count++
		c.snapshotEncode.sum += duration
		c.snapshotBytes += encodedBytes
	}
	c.mu.Unlock()
}

func (c *Collector) RecordWebSocketEnqueue(duration time.Duration) {
	c.mu.Lock()
	c.websocketEnqueue.count++
	c.websocketEnqueue.sum += duration
	c.mu.Unlock()
}

func (c *Collector) RecordRoomQueueDepth(room string, total, maximum int) {
	c.mu.Lock()
	gauge := c.rooms[room]
	gauge.queueDepth = total
	gauge.queueMax = maximum
	c.rooms[room] = gauge
	c.mu.Unlock()
}

func (c *Collector) RecordSnapshotReplaced() {
	c.mu.Lock()
	c.snapshotReplaced++
	c.mu.Unlock()
}

func (c *Collector) RecordCriticalQueueFailure() {
	c.mu.Lock()
	c.criticalQueueFailures++
	c.mu.Unlock()
}

func (c *Collector) RemoveRoom(room string) {
	c.mu.Lock()
	delete(c.rooms, room)
	c.mu.Unlock()
}

func (c *Collector) Render(activeRooms int) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var builder strings.Builder
	metricGauge(&builder, "survive_bro_active_rooms", "Current rooms", float64(activeRooms))
	players, monsters, projectiles, pickups, queueDepth, queueMax := 0, 0, 0, 0, 0, 0
	for _, gauge := range c.rooms {
		players += gauge.players
		monsters += gauge.monsters
		projectiles += gauge.projectiles
		pickups += gauge.pickups
		queueDepth += gauge.queueDepth
		queueMax = max(queueMax, gauge.queueMax)
	}
	metricGauge(&builder, "survive_bro_players", "Players in active simulations", float64(players))
	metricGauge(&builder, "survive_bro_monsters", "Monsters in active simulations", float64(monsters))
	metricGauge(&builder, "survive_bro_projectiles", "Projectiles in active simulations", float64(projectiles))
	metricGauge(&builder, "survive_bro_pickups", "Pickups in active simulations", float64(pickups))
	metricGauge(&builder, "survive_bro_websocket_queue_depth", "Total queued WebSocket messages", float64(queueDepth))
	metricGauge(&builder, "survive_bro_websocket_queue_max", "Maximum current per-client queue depth", float64(queueMax))

	builder.WriteString("# HELP survive_bro_tick_duration_seconds Authoritative room tick duration\n# TYPE survive_bro_tick_duration_seconds histogram\n")
	cumulative := uint64(0)
	for index, upper := range tickBuckets {
		cumulative += c.tickBuckets[index]
		fmt.Fprintf(&builder, "survive_bro_tick_duration_seconds_bucket{le=\"%g\"} %d\n", upper, cumulative)
	}
	cumulative += c.tickBuckets[len(tickBuckets)]
	fmt.Fprintf(&builder, "survive_bro_tick_duration_seconds_bucket{le=\"+Inf\"} %d\n", cumulative)
	fmt.Fprintf(&builder, "survive_bro_tick_duration_seconds_sum %.9f\nsurvive_bro_tick_duration_seconds_count %d\n", c.tickSum.Seconds(), c.tickCount)

	phaseNames := make([]string, 0, len(c.phases))
	for name := range c.phases {
		phaseNames = append(phaseNames, name)
	}
	sort.Strings(phaseNames)
	builder.WriteString("# HELP survive_bro_simulation_phase_seconds Simulation phase duration\n# TYPE survive_bro_simulation_phase_seconds summary\n")
	for _, name := range phaseNames {
		aggregate := c.phases[name]
		fmt.Fprintf(&builder, "survive_bro_simulation_phase_seconds_sum{phase=\"%s\"} %.9f\n", name, aggregate.sum.Seconds())
		fmt.Fprintf(&builder, "survive_bro_simulation_phase_seconds_count{phase=\"%s\"} %d\n", name, aggregate.count)
	}
	metricCounter(&builder, "survive_bro_collision_candidate_pairs_total", "Projectile and target candidate pairs", c.candidatePairs)
	metricCounter(&builder, "survive_bro_collision_narrow_checks_total", "Exact collision checks", c.narrowChecks)
	metricCounter(&builder, "survive_bro_collision_confirmed_total", "Confirmed projectile hits", c.confirmedHits)
	durationMetric(&builder, "survive_bro_snapshot_build_seconds", "Snapshot construction duration", c.snapshotBuild)
	durationMetric(&builder, "survive_bro_snapshot_encode_seconds", "Snapshot binary encoding duration", c.snapshotEncode)
	metricCounter(&builder, "survive_bro_snapshot_encoded_bytes_total", "Encoded snapshot bytes", c.snapshotBytes)
	metricCounter(&builder, "survive_bro_websocket_encoded_bytes_total", "All encoded WebSocket bytes", c.websocketBytes)
	durationMetric(&builder, "survive_bro_websocket_enqueue_seconds", "Room WebSocket enqueue duration", c.websocketEnqueue)
	metricCounter(&builder, "survive_bro_snapshot_replaced_total", "Snapshots dropped because a client queue was full", c.snapshotReplaced)
	metricCounter(&builder, "survive_bro_critical_queue_failures_total", "Critical messages rejected by full client queues", c.criticalQueueFailures)
	return builder.String()
}

func metricGauge(builder *strings.Builder, name, help string, value float64) {
	fmt.Fprintf(builder, "# HELP %s %s\n# TYPE %s gauge\n%s %g\n", name, help, name, name, value)
}

func metricCounter(builder *strings.Builder, name, help string, value uint64) {
	fmt.Fprintf(builder, "# HELP %s %s\n# TYPE %s counter\n%s %d\n", name, help, name, name, value)
}

func durationMetric(builder *strings.Builder, name, help string, aggregate durationAggregate) {
	fmt.Fprintf(builder, "# HELP %s %s\n# TYPE %s summary\n%s_sum %.9f\n%s_count %d\n", name, help, name, name, aggregate.sum.Seconds(), name, aggregate.count)
}

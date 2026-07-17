# Level Framework

Level definitions are loaded at backend startup from `game-data/game.json`. A definition contains:

- Stable ID and display name.
- Match duration.
- Terrain and obstacle asset IDs.
- Authoritative obstacle layout.
- Enemy definitions with HP, speed, collision radius, contact damage/cooldown, and XP.
- Initial normal enemy type.
- Ordered timed events.

Supported event types are deliberately small:

- `spawn_rate`: replaces rate, maximum living count, and weighted normal-enemy composition independently.
- `monster_buff`: multiplies enemy health and movement speed for living and future enemies.
- `meteor_shower`: creates server-authoritative marked impact zones for a configured duration, warning time, rate, damage cadence, and lingering range.
- `treasure_rate`: changes the authoritative normal power-crate cadence in kills. Guaranteed non-ending boss crates remain independent.
- `spell_chest`: spawns a spell chest at a validated world position.
- `boss`: spawns one configured boss enemy, may apply per-stat positive multipliers, and may end the match when that exact boss instance dies. Boss multipliers compose with global monster buffs already active.
- `end`: resolves the match and score.

Every event may set `show`. Omitted values default to `true`; `show: false` keeps the event authoritative while excluding it from `match_started` and the client timeline. Use visible events for dramatic beats and hidden events for mechanical tuning such as spawn, health, and treasure-cadence ramps.

The create-room screen loads options from `GET /api/v1/levels` and sends the selected `levelId`. Rooms retain that definition for their lifetime, late joiners enter the same timeline, `match_started.mapId` carries the selected level ID, and snapshots carry each monster `typeId` so the client can select the correct visual.

## Level 1: Slime Meadow

| Time | Event |
| ---: | --- |
| `0:00` | Meadow Erupts: Stage 1 at 1.2/sec, cap 55; normal crates every 24 kills. |
| `1:00` | Hidden ramp: 1.8/sec, cap 85, first Greater Slimes. |
| `2:00` | Pressure Rising: enemies reach `1.5×` HP; 2.6/sec, cap 120. |
| `2:30` | First visible spell chest at the map centre. |
| `3:00` | Vanguard Slime King: effective HP about `18,000`, plus the active swarm. |
| `3:30` | Hidden ramp: 3.3/sec, cap 155; crates every 22 kills. |
| `4:00` | Enemy health reaches about `2.18×` base. |
| `5:00` | Burning Horde: 4.1/sec, cap 190, with a one-minute meteor front. |
| `6:30` | Second visible spell chest. |
| `7:00` | Warlord Slime King: effective HP about `36,500`. |
| `7:30` | Enemy health reaches about `3.05×`; 4.8/sec, cap 220; crates every 20 kills. |
| `9:00` | Overrun: 5.6/sec, cap 240, with a faster 75-second meteor storm. |
| `10:00` | Enemy health reaches about `4.26×` base. |
| `10:30` | Final visible spell chest. |
| `11:00` | Royal Twins: two kings at roughly `61,000` HP each. |
| `11:30` | Royal Deluge: 6.4/sec, cap 250. |
| `12:00` | Enemy health reaches about `5.75×`; crates every 18 kills. |
| `12:30` | Meteor Cataclysm: 90 seconds at 0.6 meteors/sec. |
| `13:00` | Final Flood: 7.2/sec, cap 260, entirely Greater Slimes. |
| `13:30` | Enemy health reaches about `7.48×`; crates every 16 kills. |
| `14:00` | Sovereign Slime King: effective HP about `269,000`; defeating it ends the level. |
| `15:00` | Fixed fallback victory. |

Spawn rates and living caps are base values before the existing player-count multiplier. At six players, the final cap can approach `975`; treat this as an authored target, not verified capacity evidence.

Level 1 uses the three existing terrain variants and three large-rock variants. Add future levels under the runtime `levels` map in `game-data/game.json`; startup validation loads them into `AvailableLevels`, and the room selector plus public room metadata use their stable IDs.

## Boss Damage Lab

`test-boss` is a manual-development level exposed through the normal level selector. It lasts ten minutes and schedules one Slime King boss at `0:00` with a `1000×` health multiplier. A concurrent spawn-rate event continuously mixes Stage 1 and Stage 2 Slimes at `1.2/sec` with base cap `35`, allowing boss and swarm behavior to be inspected together. A zero-time boss event waits until at least one living player exists before it is consumed, guaranteeing the boss actually spawns at match entry. The test-only **Auto level up** HUD action advances the shared level and opens the same synchronized three-card reward phase used by earned levels.

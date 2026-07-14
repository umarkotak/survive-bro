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
- `boss`: spawns one configured boss enemy, may apply per-stat positive multipliers, and may end the match when that exact boss instance dies. Boss multipliers compose with global monster buffs already active.
- `end`: resolves the match and score.

The create-room screen loads options from `GET /api/v1/levels` and sends the selected `levelId`. Rooms retain that definition for their lifetime, late joiners enter the same timeline, `match_started.mapId` carries the selected level ID, and snapshots carry each monster `typeId` so the client can select the correct visual.

## Level 1: Slime Meadow

| Time | Event |
| ---: | --- |
| `0:00` | Opening: Stage 1 at 0.8/sec, cap 40, giving players room to earn early upgrades. |
| `1:30` | First Ripples: 1.2/sec, cap 70, 80% Stage 1 and 20% Stage 2. |
| `3:00` | Greater Tide: 1.8/sec, cap 100, equal Stage 1/Stage 2 mix. |
| `4:30` | Hardened Gel: enemies gain `1.2×` HP and `1.08×` speed. |
| `5:00` | Vanguard Slime King: HP `×1.5`, damage `×1.15`; victory guarantees a treasure chest. |
| `6:00` | Royal Retinue: 2.6/sec, cap 140, 80% Stage 2. |
| `8:00` | Burning Horde: 3.4/sec, cap 180, plus a 90-second meteor shower at 0.35/sec. |
| `10:00` | Warlord Slime King: HP `×2.5`, damage `×1.4`, speed `×1.12`; victory guarantees a treasure chest. Enemies also gain `1.25×` HP/`1.1×` speed and spawns rise to 4.2/sec, cap 220. |
| `12:00` | Chaos Tide: 5.2/sec, cap 280, plus a denser two-minute meteor shower at 0.55/sec. |
| `13:00` | Last Frenzy: enemies gain `1.2×` HP and `1.08×` speed. |
| `14:00` | Sovereign Slime King: HP `×5`, damage `×1.8`, speed `×1.25`; defeating it ends the level. |
| `15:00` | Fixed fallback end event. |

Level 1 uses the three existing terrain variants and three large-rock variants. Add future levels under the runtime `levels` map in `game-data/game.json`; startup validation loads them into `AvailableLevels`, and the room selector plus public room metadata use their stable IDs.

## Boss Damage Lab

`test-boss` is a manual-development level exposed through the normal level selector. It lasts ten minutes and schedules one Slime King boss at `0:00` with a `1000×` health multiplier. A concurrent spawn-rate event continuously mixes Stage 1 and Stage 2 Slimes at `1.2/sec` with base cap `35`, allowing boss and swarm behavior to be inspected together. A zero-time boss event waits until at least one living player exists before it is consumed, guaranteeing the boss actually spawns at match entry. The test-only **Auto level up** HUD action advances the shared level and opens the same synchronized three-card reward phase used by earned levels.

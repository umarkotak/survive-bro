# Heavy Armament UI Direction

## Identity

Survive Bro uses the **Heavy Armament** presentation layer: military machinery fused with arcane energy. Screens should feel like a battlefield command interface placed over rich game art, not a generic web dashboard.

## Visual rules

- Let supplied scene art carry the composition. Keep controls inside intentional negative space and protect important characters, enemies, and effects from overlays.
- Use near-black steel panels, thin cyan system lines, violet active-state glow, and restrained orange danger/reward accents.
- Use condensed uppercase display type for titles and actions. Keep supporting copy small, direct, and high contrast.
- Prefer clipped corners, inset borders, bolts, short labels, and strong horizontal buttons over rounded cards and pill controls.
- Reserve strong neon glow for the current or primary action. Disabled and future actions remain visible but desaturated.
- Desktop and mobile share the same hierarchy. On narrow portrait screens, preserve the scene in the upper area and move controls toward the lower reachable area.
- Motion is atmospheric, not required for comprehension. Every video background needs a still-image fallback and must respect reduced-motion preferences.
- Menu audio stays restrained. The control always reads `Sound on` or `Sound off`; sound intent defaults on, while the background video starts technically muted for autoplay compatibility and automatically unlocks during the first user gesture. Menu actions have separate short hover and click cues.
- The home account module occupies the top-right command-panel position. Its logged-out state accepts the device-local callsign; its logged-in state shows that callsign and a compact change action. The sound control stays separate at bottom-right.

## Current routes

- `/`: cinematic main menu using the Heavy Armament background and logo, plus local callsign login/account summary. `Play` opens a full-height centered lobby overlay only after a callsign exists; the logo, actions, account module, footer, and sound control hide while it is open. The modal retains equal top and bottom viewport gaps, contains its own scrolling, and never makes the backdrop or page scroll. The titleless lobby closes from its backdrop, and its create action shares the operations heading row. Lobby, create-squad, and character selection render as mutually exclusive overlay states rather than stacked backdrops; dismissing a nested state returns to the lobby. `Armory` remains visible but natively disabled and cannot be activated.
- `/lobby`: legacy direct entry normalized to the `/` lobby overlay.
- `/armory`: reserved placeholder for direct access until persistent inventory exists.

## Menu media

`apps/game/src/config/menuMedia.ts` is the single switch for main-menu media paths, video preference, default sound intent, and video volume. Menu files live under `public/assets/misc` with concise role-based filenames. The PNG renders first and remains the fallback. When video is enabled, it cross-fades in only after the browser reports it can play.

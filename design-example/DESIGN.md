# Design System Specification: The Architectural Desktop

## 1. Overview & Creative North Star

### Creative North Star: "The Nocturnal Architect"
This design system moves away from the "neon-cyber" aesthetic into a space of "High-End Technical Professionalism." It is designed for deep work and cognitive focus. We are shifting from a high-contrast terminal look to a sophisticated, editorial desktop application. 

The system breaks the "bootstrap template" look through **tonal depth** rather than structural lines. We treat the interface as a physical workspace—layers of matte-finish glass and deep architectural planes that feel engineered, not just "drawn." By prioritizing muted functional colors and intentional asymmetry in layout, we create an environment that feels authoritative, calm, and expensive.

---

## 2. Colors & Surface Philosophy

The palette is anchored in a deep, atmospheric navy. We are intentionally desaturating our functional colors to make them feel like high-precision indicators rather than glowing lights.

### Core Surface Palette
- **Background:** `#060e20` (The deep foundation).
- **Surface-Container-Low:** `#06122d` (Used for secondary sidebars).
- **Surface-Container-Highest:** `#00225a` (Used for active modules or highlighted nodes).
- **On-Surface:** `#dee5ff` (Primary technical text).
- **On-Surface-Variant:** `#91aaeb` (Muted metadata and helper text).

### The "No-Line" Rule
Traditional 1px solid borders are strictly prohibited for sectioning. Boundaries must be defined through **background color shifts**. To separate a sidebar from the main canvas, transition from `surface` to `surface-container-low`. The interface should feel like a single cohesive sculpture rather than a series of boxed-off areas.

### Glass & Gradient Layering
Floating elements (modals, context menus, tooltips) must utilize **Glassmorphism**.
- **Fill:** `surface_variant` at 60% opacity.
- **Backdrop Blur:** 12px to 20px.
- **Signature Gradient:** For primary action buttons or significant hero nodes, use a subtle linear gradient from `primary` (#7bd0ff) to `primary_container` (#004c69) at a 135-degree angle. This adds "soul" to the technical precision.

---

## 3. Typography: Editorial Technicality

We use a dual-font strategy to balance architectural structure with high-performance readability.

*   **Display & Headlines (Space Grotesk):** This is our "Branding" font. It conveys the high-tech, engineered feel. Use `display-lg` (3.5rem) and `headline-md` (1.75rem) with tighter letter-spacing (-0.02em) to create a premium, editorial impact in headers.
*   **Body & Utility (Inter):** For complex data, property panels, and code-heavy sections, we rely on Inter. Its neutral, high-legibility personality ensures that the user can scan technical data for hours without fatigue.
*   **Hierarchy as Identity:** Use a dramatic scale jump between `headline-lg` and `label-sm`. Large, thin headlines paired with tiny, wide-tracked caps labels (`label-sm` with 0.1rem tracking) create a "bespoke" look found in premium hardware interfaces.

---

## 4. Elevation & Depth: Tonal Layering

We convey hierarchy through "Tonal Stacking" rather than drop shadows.

*   **The Layering Principle:** Place a `surface-container-lowest` (#000000) element on a `surface-container-low` (#06122d) background to create a "recessed" well. Use `surface-bright` (#002867) to signify an active, "lifted" state.
*   **Ambient Shadows:** For floating dialogs only, use a "Tinted Ambient Shadow." 
    *   `box-shadow: 0 12px 40px rgba(0, 0, 0, 0.4);`
    *   Avoid grey shadows; shadows should always feel like they are "absorbing" the navy background.
*   **The Ghost Border:** If accessibility requires a container boundary, use the `outline_variant` (#2b4680) at 15% opacity. It should be felt more than seen.

---

## 5. Components

### Buttons & Inputs
- **Primary Button:** Gradient-filled (#7bd0ff to #004c69). 4px (`DEFAULT`) radius. No border.
- **Secondary Button:** `surface-container-high` fill with `primary` text.
- **Inputs:** `surface-container-lowest` background with a `ghost-border` on focus. Use `label-md` for floating labels to maintain the technical architectural look.

### Technical Nodes & Cards
- **Forbid Divider Lines:** Use `Spacing Scale 4` (0.9rem) or `6` (1.3rem) to separate content sections within a card.
- **Node Styling:** Observe the node-link graph pattern. Nodes should use `surface-variant` with a subtle 2px `outline` in the functional color (e.g., `tertiary` for successful status) but with 0px "glow." 

### Tooltips & Context Menus
- Use the **Glassmorphism** rule.
- Positioning should always be offset by `Spacing Scale 2` (0.4rem) to maintain "breathing room" between layers.

### Added Component: The "Data-Strip"
- A horizontal or vertical bar using `surface-container-highest` used to house "State" or "Live Stats" (e.g., "CPU: 5% | RAM: 2.1GB"). This should be fixed to the bottom edge, using `label-sm` for all content.

---

## 6. Do’s and Don’ts

### Do:
- **Use Intentional Asymmetry:** Align technical data to a strict grid, but allow large display headlines to break the grid slightly to feel "custom."
- **Nesting Surfaces:** Always use at least two tiers of `surface-container` to define the layout (e.g., a Sidebar in `low` and a Workspace in `default`).
- **Subtle Functional Colors:** Use `tertiary` (#9bffce) for success states, but only in small doses—as a status dot or a thin 2px bar, never as a large background fill.

### Don’t:
- **No 100% Opaque Borders:** Never use `#ffffff` or high-contrast white for borders. It breaks the "Nocturnal Architect" atmosphere.
- **No Neon Glows:** Remove the heavy outer glows from the previous iteration. If a node is active, use a subtle `surface-bright` background shift instead of a neon shadow.
- **No Default Spacing:** Avoid "even" spacing. Use the **Spacing Scale** to create a rhythm—tighter for data (1.5 - 2), looser for layout structure (10 - 16).

---
*Director's Final Note: The goal is to make the user feel like they are operating a multi-million dollar satellite terminal. Every pixel must feel calculated, every color shift intentional.*
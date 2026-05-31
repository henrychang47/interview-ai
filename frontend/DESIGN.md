---
name: Calm Interviewer
colors:
  surface: '#fbf8fd'
  surface-dim: '#dbd9de'
  surface-bright: '#fbf8fd'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#f5f2f7'
  surface-container: '#f0edf2'
  surface-container-high: '#eae7ec'
  surface-container-highest: '#e4e1e6'
  on-surface: '#1b1b1f'
  on-surface-variant: '#4a4455'
  inverse-surface: '#303034'
  inverse-on-surface: '#f2f0f5'
  outline: '#7b7487'
  outline-variant: '#ccc3d8'
  surface-tint: '#732ee4'
  primary: '#630ed4'
  on-primary: '#ffffff'
  primary-container: '#7c3aed'
  on-primary-container: '#ede0ff'
  inverse-primary: '#d2bbff'
  secondary: '#625595'
  on-secondary: '#ffffff'
  secondary-container: '#c6b7ff'
  on-secondary-container: '#524584'
  tertiary: '#4f4d5e'
  on-tertiary: '#ffffff'
  tertiary-container: '#676577'
  on-tertiary-container: '#e7e4f8'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#eaddff'
  primary-fixed-dim: '#d2bbff'
  on-primary-fixed: '#25005a'
  on-primary-fixed-variant: '#5a00c6'
  secondary-fixed: '#e7deff'
  secondary-fixed-dim: '#ccbeff'
  on-secondary-fixed: '#1e0e4e'
  on-secondary-fixed-variant: '#4a3d7c'
  tertiary-fixed: '#e4e0f5'
  tertiary-fixed-dim: '#c7c4d8'
  on-tertiary-fixed: '#1b1a29'
  on-tertiary-fixed-variant: '#464555'
  background: '#fbf8fd'
  on-background: '#1b1b1f'
  surface-variant: '#e4e1e6'
typography:
  headline-xl:
    fontFamily: Manrope
    fontSize: 40px
    fontWeight: '700'
    lineHeight: 48px
    letterSpacing: -0.02em
  headline-lg:
    fontFamily: Manrope
    fontSize: 32px
    fontWeight: '600'
    lineHeight: 40px
    letterSpacing: -0.01em
  headline-md:
    fontFamily: Manrope
    fontSize: 24px
    fontWeight: '600'
    lineHeight: 32px
  headline-sm:
    fontFamily: Manrope
    fontSize: 20px
    fontWeight: '600'
    lineHeight: 28px
  body-lg:
    fontFamily: Plus Jakarta Sans
    fontSize: 18px
    fontWeight: '400'
    lineHeight: 28px
  body-md:
    fontFamily: Plus Jakarta Sans
    fontSize: 16px
    fontWeight: '400'
    lineHeight: 24px
  body-sm:
    fontFamily: Plus Jakarta Sans
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 20px
  label-md:
    fontFamily: Plus Jakarta Sans
    fontSize: 14px
    fontWeight: '600'
    lineHeight: 16px
    letterSpacing: 0.05em
  headline-xl-mobile:
    fontFamily: Manrope
    fontSize: 32px
    fontWeight: '700'
    lineHeight: 40px
  headline-lg-mobile:
    fontFamily: Manrope
    fontSize: 26px
    fontWeight: '600'
    lineHeight: 32px
rounded:
  sm: 0.25rem
  DEFAULT: 0.5rem
  md: 0.75rem
  lg: 1rem
  xl: 1.5rem
  full: 9999px
spacing:
  base: 8px
  xs: 4px
  sm: 12px
  md: 24px
  lg: 40px
  xl: 64px
  gutter: 24px
  margin-mobile: 16px
  margin-desktop: 48px
---

## Brand & Style

The core philosophy of this design system is **Empathetic Professionalism**. Recognizing that job interviews are high-stress events, the visual language is engineered to reduce cognitive load and heart rates. We achieve this through a "Soft Minimalist" aesthetic that prioritizes breathing room, gentle transitions, and a palette that suggests competence without being cold.

The target audience consists of career-driven individuals who value clarity and support. To meet them where they are, the UI utilizes generous white space, avoids aggressive sharp corners, and leverages subtle depth to guide focus toward the most important task: preparation. The emotional response should be one of "quiet confidence"—moving the user from a state of anxiety to a state of readiness.

## Colors

The color strategy centers on a "Violet Spectrum" that transitions from the authoritative Primary Violet (#7c3aed) to the soothing Lavender Surface (#fdfaff). 

- **Primary Violet:** Used for the most critical actions and active states. It represents progress and professional growth.
- **Secondary & Tertiary:** These softer tints are used for non-critical information hierarchy and background separation, ensuring the UI feels layered rather than flat.
- **Surface:** A warm, off-white lavender base is chosen over pure white to reduce eye strain during long practice sessions and to maintain a "warm" environmental feel.
- **Success/Warning:** Use muted, low-saturation greens and ambers to provide feedback without triggering alarm.

## Typography

This design system employs a dual-font strategy to balance structure with approachability. 

**Manrope** is used for headlines. Its modern, geometric construction provides a sense of technical stability and "modern corporate" readiness. It is set with slightly tighter letter-spacing in larger sizes to create a cohesive, editorial look.

**Plus Jakarta Sans** is utilized for body copy and labels. Its softer, more rounded terminals make long blocks of text (like interview feedback) more digestible and friendly. 

For mobile devices, typography scales down to prevent overwhelming the viewport, while maintaining generous line heights to ensure readability during high-pressure scenarios (like glancing at notes).

## Layout & Spacing

The layout is built on an **8px linear scale**, ensuring mathematical harmony across all components. We use a 12-column fluid grid for desktop and a 4-column grid for mobile.

- **Negative Space:** This design system mandates "active whitespace." Elements should never feel crowded; if in doubt, increase padding to the next spacing token.
- **The Golden Gap:** Use `md` (24px) for most component internal gutters. 
- **Content Max-Width:** To keep line lengths readable for interview scripts and feedback, the main content area is capped at 1024px, centered on the screen with fluid side margins.

## Elevation & Depth

To maintain the "Calm" mood, this design system avoids heavy, dark shadows. Instead, it utilizes **Tonal Layering** and **Ambient Glows**:

1.  **Level 0 (Base):** The Lavender Surface (#fdfaff).
2.  **Level 1 (Cards):** Pure white background with a very soft, diffused shadow (Color: Primary Violet at 5% opacity, Blur: 12px, Y-offset: 4px).
3.  **Level 2 (Popovers/Modals):** Pure white background with a slightly more pronounced ambient shadow and a subtle 1px border using the Tertiary color.

Depth is also communicated through "Backdrop Blurs" (Glassmorphism) for navigation bars, allowing the content colors to bleed through softly, maintaining a sense of spatial continuity.

## Shapes

The shape language is consistently "Soft-Rounded." The 8px base radius (Level 2) is the standard for almost all containers, including buttons, input fields, and cards. 

- **Standard (8px):** Primary buttons, text inputs, and small cards.
- **Large (16px):** Main content containers and featured "Interview Prep" cards.
- **Extra Large (24px):** Large modal containers.

This consistency eliminates "visual noise" caused by varying corner radii, contributing to the overall sense of calm and order.

## Components

### Buttons
Primary buttons use the Primary Violet with white text. Hover states should not darken the color, but rather add a subtle "glow" using a shadow tint. Secondary buttons use the Tertiary background with Primary Violet text.

### Cards
Cards are the primary container for interview questions and feedback. They must feature a white background and Level 1 elevation. Internal padding should always be at least `md` (24px).

### Input Fields
Inputs use a white background with a 1px border in the Secondary color. Upon focus, the border thickens to 2px Primary Violet with a soft outer glow.

### Chips/Tags
Used for "Skills" or "Keywords." These should be pill-shaped (using `rounded-xl`) with a Tertiary background and Primary Violet text to keep them distinct but secondary in the hierarchy.

### Progress Indicators
Progress bars and circular loaders use a gradient of Secondary to Primary Violet. The motion should be a slow, "breathing" ease-in-out to maintain the calm atmosphere.
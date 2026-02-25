# NuxtUI to shadcn-vue Verification

**Purpose**: Record visual, interaction, and accessibility checks for the NuxtUI to shadcn-vue migration.
**Last Updated**: 2026-02-25

## Scope
- Web app UI migrated from NuxtUI to shadcn-vue
- Visual parity, interactions, and accessibility

## Preconditions
- App running locally with the migration branch
- Known list of routes/screens to verify

## 4-Stage Design Workflow Alignment

### Stage 1: Layout
- **What to verify**: Structure, spacing, alignment, and layout parity vs. pre-migration UI
- **Status**: Not verified
- **Notes**: Manual review required

### Stage 2: Theme
- **What to verify**: Colors, gradients, typography, and shadows match existing design
- **Status**: Not verified
- **Notes**: Manual review required

### Stage 3: Animation
- **What to verify**: Hover, focus, and motion timing match prior behavior
- **Status**: Not verified
- **Notes**: Manual review required

### Stage 4: Implementation
- **What to verify**: Component behavior parity and no UI regressions
- **Status**: Not verified
- **Notes**: Manual review required

## Visual Checks (Responsive)

| Breakpoint | Result | Notes |
| --- | --- | --- |
| 375px | Not run | Manual verification required |
| 768px | Not run | Manual verification required |
| 1024px | Not run | Manual verification required |
| 1440px | Not run | Manual verification required |

## Interaction Checks
- Buttons, links, and toggles: Not run
- Form inputs (focus/blur/error states): Not run
- Dropdowns, popovers, and dialogs: Not run
- Navigation interactions (menu, sidebar, breadcrumbs): Not run
- Hover states and active states: Not run

## Accessibility Checks
- Keyboard navigation (Tab/Shift+Tab): Not run
- Focus visibility and focus order: Not run
- ARIA labels/roles for interactive elements: Not run
- Color contrast (WCAG AA): Not run
- Touch target size (>= 44x44px): Not run

## Responsive Confirmation
- **Status**: Not verified
- **Notes**: Requires manual review at 375/768/1024/1440px

## Blockers
- Manual QA not performed yet.
- Routes/screens list required for full verification coverage.

## Next Actions
- Run the app locally and complete the checks above.
- Capture any regressions with screenshots and exact component names.

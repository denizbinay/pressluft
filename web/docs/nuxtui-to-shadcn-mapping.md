# NuxtUI -> shadcn-vue Mapping (Pressluft)

## Scope and Constraints
- Preserve existing visual design, spacing, gradients, typography, layout, and interactions.
- Replace NuxtUI components with shadcn-vue or local wrappers without changing markup output.
- Maintain accessibility and responsive behavior at 375px, 768px, 1024px, and 1440px.

## Inventory (NuxtUI Usage in web/app)

### Layouts and Pages
- `web/app/layouts/default.vue`: UDashboardGroup, UDashboardSidebar, UNavigationMenu, UDashboardSearch, UDashboardPanel, UDashboardNavbar, UDashboardSidebarCollapse, UButton, UIcon

### Components
- `web/app/components/UserPanel.vue`: UDropdownMenu, UButton, UAvatar, UIcon, DropdownMenuItem type

### UI Wrapper Components (Currently NuxtUI-backed)
- `web/app/components/ui/UiButton.vue`: UButton
- `web/app/components/ui/UiCard.vue`: UCard
- `web/app/components/ui/UiInput.vue`: UInput
- `web/app/components/ui/UiSelect.vue`: USelect
- `web/app/components/ui/UiTextarea.vue`: UTextarea
- `web/app/components/ui/UiProgressBar.vue`: UProgress
- `web/app/components/ui/UiBadge.vue`: UBadge
- `web/app/components/ui/UiModal.vue`: UModal
- `web/app/components/ui/UiToggle.vue`: USwitch

### Styling and Config
- `web/app/assets/css/main.css`: `@import "@nuxt/ui"`
- `web/app/app.config.ts`: `defineAppConfig({ ui: { primary, gray } })`

## Mapping: NuxtUI -> shadcn-vue (with Wrapper Plan)

| NuxtUI usage | Locations | shadcn-vue replacement | Wrapper plan / notes |
| --- | --- | --- | --- |
| `UButton` | `web/app/layouts/default.vue`, `web/app/components/UserPanel.vue`, `web/app/components/ui/UiButton.vue` | `Button` | Keep `UiButton` API and slot structure; map legacy `variant`/`size` to shadcn variants and sizes while preserving existing Tailwind classes. |
| `UInput` | `web/app/components/ui/UiInput.vue` | `Input` | Keep `UiInput` public props and error styling; translate `ui` class overrides to wrapper-level classes. |
| `USelect` | `web/app/components/ui/UiSelect.vue` | `Select` or `Native Select` | Use `Select` for custom dropdown, `Native Select` if keyboard UX matches current; keep `options` mapping and label layout. |
| `UTextarea` | `web/app/components/ui/UiTextarea.vue` | `Textarea` | Mirror `rows`, placeholder, and label layout; preserve border/hover classes. |
| `USwitch` | `web/app/components/ui/UiToggle.vue` | `Switch` | Keep `UiToggle` API (`modelValue`, `label`, `disabled`), preserve label placement and color tokens. |
| `UBadge` | `web/app/components/ui/UiBadge.vue` | `Badge` | Map variants to shadcn `variant` and `class` overrides; keep size props. |
| `UProgress` | `web/app/components/ui/UiProgressBar.vue` | `Progress` | Preserve `showLabel` layout and percentage calculation; map size and color tokens. |
| `UCard` | `web/app/components/ui/UiCard.vue` | `Card` | Keep `hoverable` behavior and Tailwind class set on header/body/footer. |
| `UModal` | `web/app/components/ui/UiModal.vue` | `Dialog` | Wrap `Dialog`, `DialogContent`, `DialogHeader`, `DialogFooter` to keep `open`, `title`, and `footer` slot behavior. |
| `UDropdownMenu` | `web/app/components/UserPanel.vue` | `Dropdown Menu` | Replace `items` data structure with local adapter that matches current `DropdownMenuItem` shape. |
| `UAvatar` | `web/app/components/UserPanel.vue` | `Avatar` | Keep size and alt handling; map to `Avatar`, `AvatarImage`, `AvatarFallback`. |
| `UIcon` | `web/app/layouts/default.vue`, `web/app/components/UserPanel.vue` | No direct shadcn-vue component | Provide local `UiIcon` wrapper that preserves `name` prop (using current icon system) to avoid template churn. |
| `UNavigationMenu` | `web/app/layouts/default.vue` | `Navigation Menu` | Build vertical layout via shadcn `NavigationMenu` plus Tailwind layout classes; maintain `items` and `highlight` behavior via adapter. |
| `UDashboardGroup` | `web/app/layouts/default.vue` | No direct equivalent | Create local `UDashboardGroup` wrapper using `Sidebar` + layout container to preserve slot contract. |
| `UDashboardSidebar` | `web/app/layouts/default.vue` | `Sidebar` + `Resizable` | Implement local wrapper that keeps `v-model:open`, `collapsible`, `resizable`, and `ui` slot class overrides. |
| `UDashboardPanel` | `web/app/layouts/default.vue` | Layout container | Local wrapper for main content area; keep `grow` behavior and header/body slots. |
| `UDashboardNavbar` | `web/app/layouts/default.vue` | Custom header + `Separator` | Local wrapper for topbar slots (`leading`, `right`), with no visual changes. |
| `UDashboardSidebarCollapse` | `web/app/layouts/default.vue` | `Button` | Local wrapper uses `Button` + icon, keeps click behavior. |
| `UDashboardSearch` | `web/app/layouts/default.vue` | `Dialog` + `Command` | Rebuild command palette using `Dialog` and `Command`, preserve `groups` structure and open state. |
| `@nuxt/ui` CSS import | `web/app/assets/css/main.css` | Remove | Ensure shadcn-vue styles are in place and existing Tailwind layers remain intact. |
| `defineAppConfig.ui` | `web/app/app.config.ts` | Replace with Tailwind/CSS variables | Port primary/gray tokens into existing CSS theme or shadcn theme tokens; no visual changes. |
| `DropdownMenuItem` type | `web/app/components/UserPanel.vue` | Local type | Replace with local type that matches current data shape and shadcn dropdown menu adapter. |

## Wrapper Plan (API Compatibility)

### Wrapper goals
- Keep existing component names where templates already use `U*` (minimize template changes).
- Preserve public props and slots for `Ui*` components.
- Mirror NuxtUI variant semantics (`solid`, `outline`, `ghost`, `soft`, `subtle`) in shadcn variants/classes.
- Keep Tailwind utility classes that control the current theme.

### Wrapper placement
- Use `web/app/components/ui/` for `Ui*` wrappers (already established).
- Add `web/app/components/nuxt-compat/` (or similar) for `UDashboard*`, `UNavigationMenu`, `UDropdownMenu`, `UIcon` compatibility components so templates do not change.
- Keep shadcn-vue imports isolated inside wrappers; only wrappers reference shadcn-vue primitives.

### Wrapper details (key behavior to preserve)
- `UDashboardSidebar`: `v-model:open`, `collapsible`, `resizable`, slot class overrides (`ui.body`, `ui.footer`).
- `UDashboardPanel`: `grow` prop and header/body slots.
- `UDashboardSearch`: `groups` prop shape, open/close keyboard interaction, and command palette layout.
- `UNavigationMenu`: `items` structure, vertical layout, active highlight.
- `UDropdownMenu`: `items` structure including labels, icons, and separators.
- `UiInput`, `UiSelect`, `UiTextarea`: label layout, error display, focus ring styling.

## Replacement Batches (Component, Layout, Page)

### Batch 1: Primitive UI wrappers (low risk)
- Replace `UiButton`, `UiInput`, `UiSelect`, `UiTextarea`, `UiBadge`, `UiProgressBar`, `UiToggle`, `UiModal`, `UiCard` with shadcn-vue primitives.
- No template changes expected; only wrapper internals updated.

### Batch 2: Menus and popovers (medium risk)
- Replace `UDropdownMenu` in `web/app/components/UserPanel.vue` with shadcn `Dropdown Menu`.
- Add `UiIcon` wrapper if needed for icon parity.

### Batch 3: Navigation and sidebar layout (medium/high risk)
- Implement `UNavigationMenu` wrapper using shadcn `Navigation Menu` + Tailwind layout classes.
- Implement `UDashboardSidebar`, `UDashboardGroup`, and `UDashboardPanel` wrappers using `Sidebar` and layout containers.

### Batch 4: Command palette (high risk)
- Replace `UDashboardSearch` with `Dialog` + `Command` combo.
- Preserve `groups` data structure and keyboard navigation.

### Batch 5: Cleanup and removal
- Remove `@nuxt/ui` from `web/app/assets/css/main.css` and `web/package.json`.
- Remove NuxtUI module from `web/nuxt.config.ts`.
- Replace `defineAppConfig.ui` usage with shadcn theme tokens or CSS variables.

## 4-Stage Design Workflow (Applied to Migration)

### 1) Layout
- Identify layout containers (`UDashboardGroup`, `UDashboardSidebar`, `UDashboardPanel`, `UDashboardNavbar`) and keep the existing structure and slot boundaries.
- Ensure the sidebar/content split and topbar placement remain unchanged.

### 2) Theme
- Preserve existing Tailwind tokens and CSS variables in `web/app/assets/css/main.css`.
- Carry forward NuxtUI `ui` slot classes into shadcn wrapper class names.
- Keep `primary` and `gray` token usage consistent with current design.

### 3) Animation
- Maintain current transitions on buttons, cards, hover states, and the sidebar collapse behavior.
- Preserve command palette open/close behavior and focus-visible ring styling.

### 4) Implementation
- Replace NuxtUI components in wrappers first, then in layout components.
- Use shadcn-vue primitives and local wrappers to preserve existing template APIs.
- Verify parity at 375px, 768px, 1024px, and 1440px after each batch.

## Responsive and Accessibility Verification
- Validate layout, nav, and command palette at 375px, 768px, 1024px, 1440px.
- Confirm focus rings and keyboard navigation remain intact (dropdowns, command palette, sidebar collapse).
- Ensure touch targets remain >= 44px where applicable.

## Blockers / Open Questions
- None identified yet. If NuxtUI components rely on hidden internal behaviors, document and replicate them in wrappers during implementation.

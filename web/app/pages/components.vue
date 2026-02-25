<script setup lang="ts">
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardFooter, CardHeader } from "@/components/ui/card"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Progress } from "@/components/ui/progress"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectItemText,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Spinner } from "@/components/ui/spinner"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { cn } from "@/lib/utils"

// ── Reactive state for interactive demos ──
const inputValue = ref('')
const textareaValue = ref('')
const selectValue = ref('')
const toggleA = ref(true)
const toggleB = ref(false)
const loadingBtn = ref(false)

const selectOptions = [
  { label: 'us-east-1', value: 'us-east-1' },
  { label: 'eu-west-1', value: 'eu-west-1' },
  { label: 'ap-southeast-1', value: 'ap-southeast-1' },
]

// Modal
const { isOpen: modalOpen, open: openModal, close: closeModal } = useModal()

// Simulate loading
const simulateLoading = () => {
  loadingBtn.value = true
  setTimeout(() => { loadingBtn.value = false }, 2000)
}

const cardBaseClass =
  "rounded-xl border border-border/60 bg-card/50 backdrop-blur-sm py-0 shadow-none"

const cardHoverClass =
  "transition-all duration-200 hover:border-border/80 hover:bg-card/70 hover:shadow-lg hover:shadow-black/20 cursor-pointer"

const buttonBaseClass =
  "rounded-lg font-medium transition-colors focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background"

const fieldClass =
  "w-full rounded-lg border border-border/60 bg-background/60 px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background"

const selectTriggerClass = cn(
  fieldClass,
  "hover:border-border data-[placeholder]:text-muted-foreground",
)

const inputClass = cn(
  fieldClass,
  "hover:border-border",
)

const textareaClass = cn(
  fieldClass,
  "hover:border-border min-h-[96px]",
)

const switchClass = cn(
  "h-6 w-11 rounded-full border transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background",
  "data-[state=checked]:border-ring/60 data-[state=checked]:bg-ring/30 data-[state=unchecked]:border-border/60 data-[state=unchecked]:bg-muted/60",
  "disabled:cursor-not-allowed disabled:opacity-60",
  "[&_[data-slot=switch-thumb]]:h-5 [&_[data-slot=switch-thumb]]:w-5 [&_[data-slot=switch-thumb]]:bg-background",
)

type BadgeVariant = "default" | "success" | "warning" | "danger" | "info"

const badgeClass = (variant: BadgeVariant) => {
  const mapping: Record<BadgeVariant, string> = {
    default: "border-border/60 bg-muted/60 text-foreground",
    success: "border-primary/30 bg-primary/10 text-primary",
    warning: "border-accent/30 bg-accent/10 text-accent",
    danger: "border-destructive/30 bg-destructive/10 text-destructive",
    info: "border-accent/30 bg-accent/10 text-accent",
  }

  return cn("px-2.5 py-1 text-sm", mapping[variant])
}

type ProgressVariant = "accent" | "success" | "warning" | "danger"
type ProgressSize = "sm" | "md" | "lg"

const progressIndicatorClass = (variant: ProgressVariant) => {
  const mapping: Record<ProgressVariant, string> = {
    accent: "[&_[data-slot=progress-indicator]]:bg-accent",
    success: "[&_[data-slot=progress-indicator]]:bg-primary",
    warning: "[&_[data-slot=progress-indicator]]:bg-accent",
    danger: "[&_[data-slot=progress-indicator]]:bg-destructive",
  }

  return mapping[variant]
}

const progressHeightClass = (size: ProgressSize) => {
  const mapping: Record<ProgressSize, string> = {
    sm: "h-1.5",
    md: "h-2.5",
    lg: "h-3.5",
  }

  return mapping[size]
}

const progressPercent = (value: number, max = 100) =>
  Math.min(100, Math.max(0, (value / max) * 100))
</script>

<template>
  <div class="space-y-16">
    <!-- Page header -->
    <div>
      <h1 class="text-3xl font-semibold text-foreground">
        UI Components
      </h1>
      <p class="mt-2 text-base text-muted-foreground">
        Component library &mdash; your design system at a glance.
      </p>
    </div>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Stats row -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
      <Card
        v-for="stat in [
          { label: 'Deployments', value: '1,284', change: '+12%', variant: 'success' as const },
          { label: 'Active Services', value: '23', change: '+2', variant: 'info' as const },
          { label: 'Avg Response', value: '142ms', change: '-8ms', variant: 'success' as const },
          { label: 'Error Rate', value: '0.03%', change: '+0.01%', variant: 'warning' as const },
        ]"
        :key="stat.label"
        :class="cn(cardBaseClass, cardHoverClass)"
      >
        <CardContent class="px-6 py-5">
          <div class="flex items-start justify-between">
            <div>
              <p class="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {{ stat.label }}
              </p>
              <p class="mt-2 text-3xl font-bold text-foreground font-mono">
                {{ stat.value }}
              </p>
            </div>
            <Badge variant="outline" :class="badgeClass(stat.variant)">
              {{ stat.change }}
            </Badge>
          </div>
        </CardContent>
      </Card>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Buttons -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-6 text-xl font-semibold text-foreground">Buttons</h2>
      <Card :class="cardBaseClass">
        <CardContent class="px-6 py-5">
          <div class="space-y-6">
            <!-- Variants -->
            <div>
              <p class="mb-4 text-sm font-medium uppercase tracking-wider text-muted-foreground">Variants</p>
              <div class="flex flex-wrap items-center gap-4">
                <Button :class="cn(buttonBaseClass, 'h-10 px-4 text-sm bg-primary text-primary-foreground hover:bg-primary/90')">Primary</Button>
                <Button variant="secondary" :class="cn(buttonBaseClass, 'h-10 px-4 text-sm bg-secondary text-secondary-foreground hover:bg-secondary/80')">Secondary</Button>
                <Button variant="outline" :class="cn(buttonBaseClass, 'h-10 px-4 text-sm border-border text-foreground hover:bg-muted/60')">Outline</Button>
                <Button variant="ghost" :class="cn(buttonBaseClass, 'h-10 px-4 text-sm text-muted-foreground hover:bg-muted/50')">Ghost</Button>
                <Button variant="destructive" :class="cn(buttonBaseClass, 'h-10 px-4 text-sm bg-destructive text-destructive-foreground hover:bg-destructive/90')">Danger</Button>
              </div>
            </div>

            <!-- Sizes -->
            <div>
              <p class="mb-4 text-sm font-medium uppercase tracking-wider text-muted-foreground">Sizes</p>
              <div class="flex flex-wrap items-center gap-4">
                <Button size="sm" :class="cn(buttonBaseClass, 'h-8 px-3 text-xs')">Small</Button>
                <Button :class="cn(buttonBaseClass, 'h-10 px-4 text-sm')">Medium</Button>
                <Button :class="cn(buttonBaseClass, 'h-11 px-5 text-sm')">Large</Button>
              </div>
            </div>

            <!-- States -->
            <div>
              <p class="mb-4 text-sm font-medium uppercase tracking-wider text-muted-foreground">States</p>
              <div class="flex flex-wrap items-center gap-4">
                <Button disabled :class="cn(buttonBaseClass, 'h-10 px-4 text-sm')">Disabled</Button>
                <Button :disabled="loadingBtn" :class="cn(buttonBaseClass, 'h-10 px-4 text-sm')" @click="simulateLoading">
                  <Spinner v-if="loadingBtn" class="h-4 w-4" />
                  {{ loadingBtn ? 'Deploying...' : 'Click to load' }}
                </Button>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Badges -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-6 text-xl font-semibold text-foreground">Badges</h2>
      <Card :class="cardBaseClass">
        <CardContent class="px-6 py-5">
          <div class="flex flex-wrap items-center gap-4">
            <Badge variant="outline" :class="badgeClass('default')">Default</Badge>
            <Badge variant="outline" :class="badgeClass('success')">Healthy</Badge>
            <Badge variant="outline" :class="badgeClass('warning')">Degraded</Badge>
            <Badge variant="outline" :class="badgeClass('danger')">Down</Badge>
            <Badge variant="outline" :class="badgeClass('info')">Deploying</Badge>
          </div>
        </CardContent>
      </Card>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Progress Bars -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-6 text-xl font-semibold text-foreground">Progress Bars</h2>
      <Card :class="cardBaseClass">
        <CardContent class="px-6 py-5">
          <div class="space-y-6">
            <div class="w-full">
              <div class="mb-1.5 flex items-center justify-between">
                  <span class="text-sm font-medium text-muted-foreground">CPU Usage</span>
                  <span class="text-xs font-mono text-muted-foreground">{{ Math.round(progressPercent(72)) }}%</span>
                </div>
                <Progress
                  :model-value="progressPercent(72)"
                  :class="cn('bg-muted/70', progressHeightClass('md'), progressIndicatorClass('accent'))"
                />
              </div>
            <div class="w-full">
              <div class="mb-1.5 flex items-center justify-between">
                  <span class="text-sm font-medium text-muted-foreground">Memory</span>
                  <span class="text-xs font-mono text-muted-foreground">{{ Math.round(progressPercent(45)) }}%</span>
                </div>
                <Progress
                  :model-value="progressPercent(45)"
                  :class="cn('bg-muted/70', progressHeightClass('md'), progressIndicatorClass('success'))"
                />
              </div>
            <div class="w-full">
              <div class="mb-1.5 flex items-center justify-between">
                  <span class="text-sm font-medium text-muted-foreground">Disk</span>
                  <span class="text-xs font-mono text-muted-foreground">{{ Math.round(progressPercent(89)) }}%</span>
                </div>
                <Progress
                  :model-value="progressPercent(89)"
                  :class="cn('bg-muted/70', progressHeightClass('md'), progressIndicatorClass('warning'))"
                />
              </div>
            <Progress
              :model-value="progressPercent(12)"
              :class="cn('bg-muted/70', progressHeightClass('sm'), progressIndicatorClass('danger'))"
            />
            <Progress
              :model-value="progressPercent(60)"
              :class="cn('bg-muted/70', progressHeightClass('lg'), progressIndicatorClass('accent'))"
            />
          </div>
        </CardContent>
      </Card>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Cards -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-6 text-xl font-semibold text-foreground">Cards</h2>
      <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
        <Card :class="cardBaseClass">
          <CardHeader class="border-b border-border/40 px-6 py-5">
            <h3 class="text-base font-semibold text-foreground">Basic Card</h3>
          </CardHeader>
          <CardContent class="px-6 py-5">
            <p class="text-base text-muted-foreground">
              A standard card with header, body, and footer slots. Use it for grouping related content.
            </p>
          </CardContent>
          <CardFooter class="border-t border-border/40 px-6 py-4">
            <div class="flex justify-end">
              <Button
                variant="ghost"
                size="sm"
                :class="cn(buttonBaseClass, 'h-8 px-3 text-xs text-muted-foreground hover:bg-muted/50')"
              >
                View details
              </Button>
            </div>
          </CardFooter>
        </Card>

        <Card :class="cn(cardBaseClass, cardHoverClass)">
          <CardHeader class="border-b border-border/40 px-6 py-5">
            <div class="flex items-center justify-between">
              <h3 class="text-base font-semibold text-foreground">Hoverable Card</h3>
              <Badge variant="outline" :class="badgeClass('success')">Active</Badge>
            </div>
          </CardHeader>
          <CardContent class="px-6 py-5">
            <p class="text-base text-muted-foreground">
              This card has a hover effect. Useful for clickable items like service cards or pipeline entries.
            </p>
          </CardContent>
        </Card>
      </div>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Forms -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-6 text-xl font-semibold text-foreground">Form Controls</h2>
      <Card :class="cardBaseClass">
        <CardContent class="px-6 py-5">
          <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
            <div class="space-y-1.5">
              <Label class="text-sm font-medium text-muted-foreground">Service Name</Label>
              <Input
                v-model="inputValue"
                placeholder="e.g. api-gateway"
                :class="inputClass"
              />
            </div>
            <div class="space-y-1.5">
              <Label class="text-sm font-medium text-muted-foreground">Region</Label>
              <Select v-model="selectValue">
                <SelectTrigger :class="selectTriggerClass">
                  <SelectValue placeholder="Select an option" />
                </SelectTrigger>
                <SelectContent class="border-border/60 bg-popover text-popover-foreground">
                  <SelectItem
                    v-for="option in selectOptions"
                    :key="option.value"
                    :value="option.value"
                    class="text-foreground data-[disabled]:text-muted-foreground data-[highlighted]:bg-muted/60 data-[highlighted]:text-foreground"
                  >
                    <SelectItemText>{{ option.label }}</SelectItemText>
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div class="md:col-span-2 space-y-1.5">
              <Label class="text-sm font-medium text-muted-foreground">Description</Label>
              <Textarea
                v-model="textareaValue"
                placeholder="Describe the service configuration..."
                :class="textareaClass"
                rows="4"
              />
            </div>
            <div class="space-y-1.5">
              <Label class="text-sm font-medium text-muted-foreground">With Error</Label>
              <Input
                model-value="bad-value"
                placeholder="Invalid input"
                aria-invalid="true"
                :class="cn(inputClass, 'border-destructive/60 focus-visible:ring-destructive/40')"
              />
              <p class="text-xs text-destructive">This field is required</p>
            </div>
            <div class="space-y-1.5">
              <Label class="text-sm font-medium text-muted-foreground">Disabled</Label>
              <Input
                placeholder="Cannot edit"
                disabled
                :class="inputClass"
              />
            </div>
          </div>
        </CardContent>
      </Card>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Toggles -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-6 text-xl font-semibold text-foreground">Toggles</h2>
      <Card :class="cardBaseClass">
        <CardContent class="px-6 py-5">
          <div class="space-y-5">
            <div class="flex items-center justify-between gap-3">
              <Label class="text-sm text-foreground/80">Auto-deploy on push</Label>
              <Switch v-model:checked="toggleA" :class="switchClass" />
            </div>
            <div class="flex items-center justify-between gap-3">
              <Label class="text-sm text-foreground/80">Enable notifications</Label>
              <Switch v-model:checked="toggleB" :class="switchClass" />
            </div>
            <div class="flex items-center justify-between gap-3">
              <Label class="text-sm text-muted-foreground">Disabled toggle</Label>
              <Switch disabled :class="switchClass" />
            </div>
          </div>
        </CardContent>
      </Card>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Dropdown -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-6 text-xl font-semibold text-foreground">Dropdown</h2>
      <Card :class="cardBaseClass">
        <CardContent class="px-6 py-5">
          <div class="flex gap-5">
            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <Button
                  variant="secondary"
                  :class="cn(buttonBaseClass, 'h-10 px-4 text-sm bg-secondary text-secondary-foreground hover:bg-secondary/80')"
                >
                  Actions
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
                  </svg>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent :class="cn('rounded-lg border border-border bg-popover p-1 shadow-xl text-popover-foreground')">
                <DropdownMenuItem class="text-foreground hover:bg-muted focus:bg-muted">Deploy</DropdownMenuItem>
                <DropdownMenuItem class="text-foreground hover:bg-muted focus:bg-muted">Rollback</DropdownMenuItem>
                <DropdownMenuItem class="text-foreground hover:bg-muted focus:bg-muted">View Logs</DropdownMenuItem>
                <DropdownMenuSeparator class="mx-0 my-2 h-px bg-border/40" />
                <DropdownMenuItem class="text-destructive hover:bg-destructive/10 hover:text-destructive focus:bg-destructive/10 focus:text-destructive">
                  Delete Service
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <Button
                  variant="outline"
                  :class="cn(buttonBaseClass, 'h-10 px-4 text-sm border-border text-foreground hover:bg-muted/60')"
                >
                  Options
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
                  </svg>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" :class="cn('rounded-lg border border-border bg-popover p-1 shadow-xl text-popover-foreground')">
                <DropdownMenuItem class="text-foreground hover:bg-muted focus:bg-muted">Edit</DropdownMenuItem>
                <DropdownMenuItem class="text-foreground hover:bg-muted focus:bg-muted">Duplicate</DropdownMenuItem>
                <DropdownMenuItem disabled class="text-muted-foreground">Archive</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </CardContent>
      </Card>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Modal -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-6 text-xl font-semibold text-foreground">Modal</h2>
      <Card :class="cardBaseClass">
        <CardContent class="px-6 py-5">
          <Button :class="cn(buttonBaseClass, 'h-10 px-4 text-sm bg-primary text-primary-foreground hover:bg-primary/90')" @click="openModal">
            Open Modal
          </Button>
        </CardContent>
      </Card>

      <Dialog :open="modalOpen" @update:open="(value) => (value ? openModal() : closeModal())">
        <DialogContent
          :show-close-button="false"
          class="w-full max-w-lg rounded-xl border border-border/60 bg-popover/90 p-0 shadow-2xl text-popover-foreground"
        >
          <div class="flex items-center justify-between border-b border-border/40 px-6 py-4">
            <DialogTitle class="text-base font-semibold text-foreground">Confirm Deployment</DialogTitle>
            <DialogClose
              class="inline-flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground transition hover:bg-muted/60 hover:text-foreground"
              aria-label="Close modal"
            >
              <span aria-hidden="true">×</span>
            </DialogClose>
          </div>

          <div class="px-6 py-5">
            <p class="text-base text-muted-foreground">
              You are about to deploy <span class="font-mono text-foreground">api-gateway@v2.4.1</span>
              to <span class="font-semibold text-foreground">production</span>. This action will
              replace the currently running version.
            </p>
          </div>
          <DialogFooter class="border-t border-border/40 px-6 py-4">
            <div class="flex justify-end gap-4">
              <Button
                variant="ghost"
                :class="cn(buttonBaseClass, 'h-10 px-4 text-sm text-muted-foreground hover:bg-muted/50')"
                @click="closeModal"
              >
                Cancel
              </Button>
              <Button
                :class="cn(buttonBaseClass, 'h-10 px-4 text-sm bg-primary text-primary-foreground hover:bg-primary/90')"
                @click="closeModal"
              >
                Deploy
              </Button>
            </div>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Typography & Colors -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-6 text-xl font-semibold text-foreground">Typography &amp; Colors</h2>
      <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
        <Card :class="cardBaseClass">
          <CardHeader class="border-b border-border/40 px-6 py-5">
            <h3 class="text-base font-semibold text-foreground">Type Scale</h3>
          </CardHeader>
          <CardContent class="px-6 py-5">
            <div class="space-y-4">
              <p class="text-4xl font-bold text-foreground">Heading 1</p>
              <p class="text-3xl font-semibold text-foreground">Heading 2</p>
              <p class="text-2xl font-semibold text-foreground">Heading 3</p>
              <p class="text-lg text-foreground/80">Body text — Inter 400</p>
              <p class="text-base text-muted-foreground">Secondary text — Inter 400</p>
              <p class="text-sm text-muted-foreground">Caption text — Inter 400</p>
              <p class="font-mono text-base text-accent">Monospace — JetBrains Mono</p>
            </div>
          </CardContent>
        </Card>

        <Card :class="cardBaseClass">
          <CardHeader class="border-b border-border/40 px-6 py-5">
            <h3 class="text-base font-semibold text-foreground">Color Palette</h3>
          </CardHeader>
          <CardContent class="px-6 py-5">
            <div class="space-y-5">
              <div>
                <p class="mb-2 text-sm font-medium text-muted-foreground">Surface</p>
                <div class="flex gap-1.5">
                  <div class="h-10 w-10 rounded-lg bg-background" title="background" />
                  <div class="h-10 w-10 rounded-lg bg-card" title="card" />
                  <div class="h-10 w-10 rounded-lg bg-muted" title="muted" />
                  <div class="h-10 w-10 rounded-lg bg-secondary" title="secondary" />
                  <div class="h-10 w-10 rounded-lg bg-border" title="border" />
                  <div class="h-10 w-10 rounded-lg bg-foreground" title="foreground" />
                  <div class="h-10 w-10 rounded-lg bg-input" title="input" />
                  <div class="h-10 w-10 rounded-lg bg-popover" title="popover" />
                </div>
              </div>
              <div>
                <p class="mb-2 text-sm font-medium text-muted-foreground">Accent</p>
                <div class="flex gap-1.5">
                  <div class="h-10 w-10 rounded-lg bg-accent" title="accent" />
                  <div class="h-10 w-10 rounded-lg bg-primary" title="primary" />
                  <div class="h-10 w-10 rounded-lg bg-ring" title="ring" />
                  <div class="h-10 w-10 rounded-lg bg-destructive" title="destructive" />
                  <div class="h-10 w-10 rounded-lg bg-muted-foreground" title="muted-foreground" />
                </div>
              </div>
              <div>
                <p class="mb-2 text-sm font-medium text-muted-foreground">Semantic</p>
                <div class="flex gap-4">
                  <div class="flex items-center gap-2">
                    <div class="h-8 w-8 rounded-lg bg-primary" />
                    <span class="text-sm text-muted-foreground">Primary</span>
                  </div>
                  <div class="flex items-center gap-2">
                    <div class="h-8 w-8 rounded-lg bg-accent" />
                    <span class="text-sm text-muted-foreground">Accent</span>
                  </div>
                  <div class="flex items-center gap-2">
                    <div class="h-8 w-8 rounded-lg bg-destructive" />
                    <span class="text-sm text-muted-foreground">Destructive</span>
                  </div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Glass effect demo -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-6 text-xl font-semibold text-foreground">Special Effects</h2>
      <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
        <div class="glass rounded-xl p-6">
          <h3 class="text-base font-semibold text-foreground">Glass Effect</h3>
          <p class="mt-2 text-base text-muted-foreground">
            Frosted glass panel with backdrop blur. Use for overlays and elevated surfaces.
          </p>
        </div>
        <div class="glow-accent rounded-xl border border-accent/20 bg-card/50 p-6">
          <h3 class="text-base font-semibold text-accent">Accent Glow</h3>
          <p class="mt-2 text-base text-muted-foreground">
            Subtle glow effect for highlighted or featured elements.
          </p>
        </div>
      </div>
    </section>
  </div>
</template>

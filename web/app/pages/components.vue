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
  "rounded-xl border border-surface-800/60 bg-surface-900/50 backdrop-blur-sm py-0 shadow-none"

const cardHoverClass =
  "transition-all duration-200 hover:border-surface-700/80 hover:bg-surface-900/70 hover:shadow-lg hover:shadow-surface-950/50 cursor-pointer"

const buttonBaseClass =
  "rounded-lg font-medium transition-colors focus-visible:ring-2 focus-visible:ring-accent-500/60 focus-visible:ring-offset-2 focus-visible:ring-offset-surface-950"

const fieldClass =
  "w-full rounded-lg border bg-surface-900/60 px-3 py-2 text-sm text-surface-100 placeholder:text-surface-400 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-500/60 focus-visible:ring-offset-2 focus-visible:ring-offset-surface-950"

const selectTriggerClass = cn(
  fieldClass,
  "border-surface-700/60 hover:border-surface-600 data-[placeholder]:text-surface-400",
)

const inputClass = cn(
  fieldClass,
  "border-surface-700/60 hover:border-surface-600",
)

const textareaClass = cn(
  fieldClass,
  "border-surface-700/60 hover:border-surface-600 min-h-[96px]",
)

const switchClass = cn(
  "h-6 w-11 rounded-full border transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-500/60 focus-visible:ring-offset-2 focus-visible:ring-offset-surface-950",
  "data-[state=checked]:border-accent-500/60 data-[state=checked]:bg-accent-500/40 data-[state=unchecked]:border-surface-700/60 data-[state=unchecked]:bg-surface-800/60",
  "disabled:cursor-not-allowed disabled:opacity-60",
  "[&_[data-slot=switch-thumb]]:h-5 [&_[data-slot=switch-thumb]]:w-5 [&_[data-slot=switch-thumb]]:bg-surface-100",
)

type BadgeVariant = "default" | "success" | "warning" | "danger" | "info"

const badgeClass = (variant: BadgeVariant) => {
  const mapping: Record<BadgeVariant, string> = {
    default: "border-surface-700/60 bg-surface-800/60 text-surface-100",
    success: "border-success-700/40 bg-success-900/40 text-success-300",
    warning: "border-warning-700/40 bg-warning-900/40 text-warning-300",
    danger: "border-danger-700/40 bg-danger-900/40 text-danger-300",
    info: "border-accent-600/40 bg-accent-500/15 text-accent-200",
  }

  return cn("px-2.5 py-1 text-sm", mapping[variant])
}

type ProgressVariant = "accent" | "success" | "warning" | "danger"
type ProgressSize = "sm" | "md" | "lg"

const progressIndicatorClass = (variant: ProgressVariant) => {
  const mapping: Record<ProgressVariant, string> = {
    accent: "[&_[data-slot=progress-indicator]]:bg-accent-500",
    success: "[&_[data-slot=progress-indicator]]:bg-success-500",
    warning: "[&_[data-slot=progress-indicator]]:bg-warning-500",
    danger: "[&_[data-slot=progress-indicator]]:bg-danger-500",
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
      <h1 class="text-3xl font-semibold text-surface-50">
        UI Components
      </h1>
      <p class="mt-2 text-base text-surface-400">
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
              <p class="text-xs font-medium uppercase tracking-wider text-surface-500">
                {{ stat.label }}
              </p>
              <p class="mt-2 text-3xl font-bold text-surface-50 font-mono">
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
      <h2 class="mb-6 text-xl font-semibold text-surface-100">Buttons</h2>
      <Card :class="cardBaseClass">
        <CardContent class="px-6 py-5">
          <div class="space-y-6">
            <!-- Variants -->
            <div>
              <p class="mb-4 text-sm font-medium uppercase tracking-wider text-surface-500">Variants</p>
              <div class="flex flex-wrap items-center gap-4">
                <Button :class="cn(buttonBaseClass, 'h-10 px-4 text-sm bg-primary text-primary-foreground hover:bg-primary/90')">Primary</Button>
                <Button variant="secondary" :class="cn(buttonBaseClass, 'h-10 px-4 text-sm bg-surface-800 text-surface-100 hover:bg-surface-700')">Secondary</Button>
                <Button variant="outline" :class="cn(buttonBaseClass, 'h-10 px-4 text-sm border-surface-700 text-surface-100 hover:bg-surface-800/60')">Outline</Button>
                <Button variant="ghost" :class="cn(buttonBaseClass, 'h-10 px-4 text-sm text-surface-200 hover:bg-surface-800/50')">Ghost</Button>
                <Button variant="destructive" :class="cn(buttonBaseClass, 'h-10 px-4 text-sm bg-danger-600 text-white hover:bg-danger-500')">Danger</Button>
              </div>
            </div>

            <!-- Sizes -->
            <div>
              <p class="mb-4 text-sm font-medium uppercase tracking-wider text-surface-500">Sizes</p>
              <div class="flex flex-wrap items-center gap-4">
                <Button size="sm" :class="cn(buttonBaseClass, 'h-8 px-3 text-xs')">Small</Button>
                <Button :class="cn(buttonBaseClass, 'h-10 px-4 text-sm')">Medium</Button>
                <Button :class="cn(buttonBaseClass, 'h-11 px-5 text-sm')">Large</Button>
              </div>
            </div>

            <!-- States -->
            <div>
              <p class="mb-4 text-sm font-medium uppercase tracking-wider text-surface-500">States</p>
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
      <h2 class="mb-6 text-xl font-semibold text-surface-100">Badges</h2>
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
      <h2 class="mb-6 text-xl font-semibold text-surface-100">Progress Bars</h2>
      <Card :class="cardBaseClass">
        <CardContent class="px-6 py-5">
          <div class="space-y-6">
            <div class="w-full">
              <div class="mb-1.5 flex items-center justify-between">
                <span class="text-sm font-medium text-surface-300">CPU Usage</span>
                <span class="text-xs font-mono text-surface-400">{{ Math.round(progressPercent(72)) }}%</span>
              </div>
              <Progress
                :model-value="progressPercent(72)"
                :class="cn('bg-surface-800/70', progressHeightClass('md'), progressIndicatorClass('accent'))"
              />
            </div>
            <div class="w-full">
              <div class="mb-1.5 flex items-center justify-between">
                <span class="text-sm font-medium text-surface-300">Memory</span>
                <span class="text-xs font-mono text-surface-400">{{ Math.round(progressPercent(45)) }}%</span>
              </div>
              <Progress
                :model-value="progressPercent(45)"
                :class="cn('bg-surface-800/70', progressHeightClass('md'), progressIndicatorClass('success'))"
              />
            </div>
            <div class="w-full">
              <div class="mb-1.5 flex items-center justify-between">
                <span class="text-sm font-medium text-surface-300">Disk</span>
                <span class="text-xs font-mono text-surface-400">{{ Math.round(progressPercent(89)) }}%</span>
              </div>
              <Progress
                :model-value="progressPercent(89)"
                :class="cn('bg-surface-800/70', progressHeightClass('md'), progressIndicatorClass('warning'))"
              />
            </div>
            <Progress
              :model-value="progressPercent(12)"
              :class="cn('bg-surface-800/70', progressHeightClass('sm'), progressIndicatorClass('danger'))"
            />
            <Progress
              :model-value="progressPercent(60)"
              :class="cn('bg-surface-800/70', progressHeightClass('lg'), progressIndicatorClass('accent'))"
            />
          </div>
        </CardContent>
      </Card>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Cards -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-6 text-xl font-semibold text-surface-100">Cards</h2>
      <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
        <Card :class="cardBaseClass">
          <CardHeader class="border-b border-surface-800/40 px-6 py-5">
            <h3 class="text-base font-semibold text-surface-100">Basic Card</h3>
          </CardHeader>
          <CardContent class="px-6 py-5">
            <p class="text-base text-surface-400">
              A standard card with header, body, and footer slots. Use it for grouping related content.
            </p>
          </CardContent>
          <CardFooter class="border-t border-surface-800/40 px-6 py-4">
            <div class="flex justify-end">
              <Button
                variant="ghost"
                size="sm"
                :class="cn(buttonBaseClass, 'h-8 px-3 text-xs text-surface-200 hover:bg-surface-800/50')"
              >
                View details
              </Button>
            </div>
          </CardFooter>
        </Card>

        <Card :class="cn(cardBaseClass, cardHoverClass)">
          <CardHeader class="border-b border-surface-800/40 px-6 py-5">
            <div class="flex items-center justify-between">
              <h3 class="text-base font-semibold text-surface-100">Hoverable Card</h3>
              <Badge variant="outline" :class="badgeClass('success')">Active</Badge>
            </div>
          </CardHeader>
          <CardContent class="px-6 py-5">
            <p class="text-base text-surface-400">
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
      <h2 class="mb-6 text-xl font-semibold text-surface-100">Form Controls</h2>
      <Card :class="cardBaseClass">
        <CardContent class="px-6 py-5">
          <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
            <div class="space-y-1.5">
              <Label class="text-sm font-medium text-surface-300">Service Name</Label>
              <Input
                v-model="inputValue"
                placeholder="e.g. api-gateway"
                :class="inputClass"
              />
            </div>
            <div class="space-y-1.5">
              <Label class="text-sm font-medium text-surface-300">Region</Label>
              <Select v-model="selectValue">
                <SelectTrigger :class="selectTriggerClass">
                  <SelectValue placeholder="Select an option" />
                </SelectTrigger>
                <SelectContent class="border-surface-800/60 bg-surface-950 text-surface-100">
                  <SelectItem
                    v-for="option in selectOptions"
                    :key="option.value"
                    :value="option.value"
                    class="text-surface-100 data-[disabled]:text-surface-500 data-[highlighted]:bg-surface-800/60 data-[highlighted]:text-surface-50"
                  >
                    <SelectItemText>{{ option.label }}</SelectItemText>
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div class="md:col-span-2 space-y-1.5">
              <Label class="text-sm font-medium text-surface-300">Description</Label>
              <Textarea
                v-model="textareaValue"
                placeholder="Describe the service configuration..."
                :class="textareaClass"
                rows="4"
              />
            </div>
            <div class="space-y-1.5">
              <Label class="text-sm font-medium text-surface-300">With Error</Label>
              <Input
                model-value="bad-value"
                placeholder="Invalid input"
                aria-invalid="true"
                :class="cn(inputClass, 'border-danger-500/60 focus-visible:ring-danger-500/40')"
              />
              <p class="text-xs text-danger-400">This field is required</p>
            </div>
            <div class="space-y-1.5">
              <Label class="text-sm font-medium text-surface-300">Disabled</Label>
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
      <h2 class="mb-6 text-xl font-semibold text-surface-100">Toggles</h2>
      <Card :class="cardBaseClass">
        <CardContent class="px-6 py-5">
          <div class="space-y-5">
            <div class="flex items-center justify-between gap-3">
              <Label class="text-sm text-surface-200">Auto-deploy on push</Label>
              <Switch v-model:checked="toggleA" :class="switchClass" />
            </div>
            <div class="flex items-center justify-between gap-3">
              <Label class="text-sm text-surface-200">Enable notifications</Label>
              <Switch v-model:checked="toggleB" :class="switchClass" />
            </div>
            <div class="flex items-center justify-between gap-3">
              <Label class="text-sm text-surface-400">Disabled toggle</Label>
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
      <h2 class="mb-6 text-xl font-semibold text-surface-100">Dropdown</h2>
      <Card :class="cardBaseClass">
        <CardContent class="px-6 py-5">
          <div class="flex gap-5">
            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <Button
                  variant="secondary"
                  :class="cn(buttonBaseClass, 'h-10 px-4 text-sm bg-surface-800 text-surface-100 hover:bg-surface-700')"
                >
                  Actions
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
                  </svg>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent :class="cn('rounded-lg border border-default bg-default p-1 shadow-xl')">
                <DropdownMenuItem class="text-foreground hover:bg-default focus:bg-default">Deploy</DropdownMenuItem>
                <DropdownMenuItem class="text-foreground hover:bg-default focus:bg-default">Rollback</DropdownMenuItem>
                <DropdownMenuItem class="text-foreground hover:bg-default focus:bg-default">View Logs</DropdownMenuItem>
                <DropdownMenuSeparator class="mx-0 my-2 h-px bg-surface-700/40" />
                <DropdownMenuItem class="text-error hover:bg-error/10 hover:text-error focus:bg-error/10 focus:text-error">
                  Delete Service
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <Button
                  variant="outline"
                  :class="cn(buttonBaseClass, 'h-10 px-4 text-sm border-surface-700 text-surface-100 hover:bg-surface-800/60')"
                >
                  Options
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
                  </svg>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" :class="cn('rounded-lg border border-default bg-default p-1 shadow-xl')">
                <DropdownMenuItem class="text-foreground hover:bg-default focus:bg-default">Edit</DropdownMenuItem>
                <DropdownMenuItem class="text-foreground hover:bg-default focus:bg-default">Duplicate</DropdownMenuItem>
                <DropdownMenuItem disabled class="text-foreground/50">Archive</DropdownMenuItem>
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
      <h2 class="mb-6 text-xl font-semibold text-surface-100">Modal</h2>
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
          class="w-full max-w-lg rounded-xl border border-surface-800/60 bg-surface-900/80 p-0 shadow-2xl"
        >
          <div class="flex items-center justify-between border-b border-surface-800/40 px-6 py-4">
            <DialogTitle class="text-base font-semibold text-surface-100">Confirm Deployment</DialogTitle>
            <DialogClose
              class="inline-flex h-8 w-8 items-center justify-center rounded-md text-surface-300 transition hover:bg-surface-800/60 hover:text-surface-100"
              aria-label="Close modal"
            >
              <span aria-hidden="true">×</span>
            </DialogClose>
          </div>

          <div class="px-6 py-5">
            <p class="text-base text-surface-400">
              You are about to deploy <span class="font-mono text-surface-200">api-gateway@v2.4.1</span>
              to <span class="font-semibold text-surface-200">production</span>. This action will
              replace the currently running version.
            </p>
          </div>
          <DialogFooter class="border-t border-surface-800/40 px-6 py-4">
            <div class="flex justify-end gap-4">
              <Button
                variant="ghost"
                :class="cn(buttonBaseClass, 'h-10 px-4 text-sm text-surface-200 hover:bg-surface-800/50')"
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
      <h2 class="mb-6 text-xl font-semibold text-surface-100">Typography &amp; Colors</h2>
      <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
        <Card :class="cardBaseClass">
          <CardHeader class="border-b border-surface-800/40 px-6 py-5">
            <h3 class="text-base font-semibold text-surface-100">Type Scale</h3>
          </CardHeader>
          <CardContent class="px-6 py-5">
            <div class="space-y-4">
              <p class="text-4xl font-bold text-surface-50">Heading 1</p>
              <p class="text-3xl font-semibold text-surface-50">Heading 2</p>
              <p class="text-2xl font-semibold text-surface-100">Heading 3</p>
              <p class="text-lg text-surface-200">Body text — Inter 400</p>
              <p class="text-base text-surface-400">Secondary text — Inter 400</p>
              <p class="text-sm text-surface-500">Caption text — Inter 400</p>
              <p class="font-mono text-base text-accent-400">Monospace — JetBrains Mono</p>
            </div>
          </CardContent>
        </Card>

        <Card :class="cardBaseClass">
          <CardHeader class="border-b border-surface-800/40 px-6 py-5">
            <h3 class="text-base font-semibold text-surface-100">Color Palette</h3>
          </CardHeader>
          <CardContent class="px-6 py-5">
            <div class="space-y-5">
              <div>
                <p class="mb-2 text-sm font-medium text-surface-500">Surface</p>
                <div class="flex gap-1.5">
                  <div class="h-10 w-10 rounded-lg bg-surface-950" title="950" />
                  <div class="h-10 w-10 rounded-lg bg-surface-900" title="900" />
                  <div class="h-10 w-10 rounded-lg bg-surface-800" title="800" />
                  <div class="h-10 w-10 rounded-lg bg-surface-700" title="700" />
                  <div class="h-10 w-10 rounded-lg bg-surface-600" title="600" />
                  <div class="h-10 w-10 rounded-lg bg-surface-500" title="500" />
                  <div class="h-10 w-10 rounded-lg bg-surface-400" title="400" />
                  <div class="h-10 w-10 rounded-lg bg-surface-300" title="300" />
                </div>
              </div>
              <div>
                <p class="mb-2 text-sm font-medium text-surface-500">Accent</p>
                <div class="flex gap-1.5">
                  <div class="h-10 w-10 rounded-lg bg-accent-700" title="700" />
                  <div class="h-10 w-10 rounded-lg bg-accent-600" title="600" />
                  <div class="h-10 w-10 rounded-lg bg-accent-500" title="500" />
                  <div class="h-10 w-10 rounded-lg bg-accent-400" title="400" />
                  <div class="h-10 w-10 rounded-lg bg-accent-300" title="300" />
                </div>
              </div>
              <div>
                <p class="mb-2 text-sm font-medium text-surface-500">Semantic</p>
                <div class="flex gap-4">
                  <div class="flex items-center gap-2">
                    <div class="h-8 w-8 rounded-lg bg-success-500" />
                    <span class="text-sm text-surface-400">Success</span>
                  </div>
                  <div class="flex items-center gap-2">
                    <div class="h-8 w-8 rounded-lg bg-warning-500" />
                    <span class="text-sm text-surface-400">Warning</span>
                  </div>
                  <div class="flex items-center gap-2">
                    <div class="h-8 w-8 rounded-lg bg-danger-500" />
                    <span class="text-sm text-surface-400">Danger</span>
                  </div>
                  <div class="flex items-center gap-2">
                    <div class="h-8 w-8 rounded-lg bg-primary-500" />
                    <span class="text-sm text-surface-400">Primary</span>
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
      <h2 class="mb-6 text-xl font-semibold text-surface-100">Special Effects</h2>
      <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
        <div class="glass rounded-xl p-6">
          <h3 class="text-base font-semibold text-surface-100">Glass Effect</h3>
          <p class="mt-2 text-base text-surface-400">
            Frosted glass panel with backdrop blur. Use for overlays and elevated surfaces.
          </p>
        </div>
        <div class="glow-accent rounded-xl border border-accent-500/20 bg-surface-900/50 p-6">
          <h3 class="text-base font-semibold text-accent-300">Accent Glow</h3>
          <p class="mt-2 text-base text-surface-400">
            Subtle glow effect for highlighted or featured elements.
          </p>
        </div>
      </div>
    </section>
  </div>
</template>

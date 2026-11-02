<script lang="ts" module>
	export type SelectItem = { value: string; label: string };
	export type SelectGroup = { label: string; items: SelectItem[] };
</script>

<script lang="ts" generics="M extends boolean = false">
	import { cn } from '$lib/utils';
	import CheckIcon from '@lucide/svelte/icons/check';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';

	type Value = M extends true ? string[] : string;

	let {
		value = $bindable(),
		multiple,
		items = [],
		groups,
		placeholder = 'Select',
		disabled,
		onchange,
		class: className
	}: {
		value: Value;
		multiple?: M;
		items?: SelectItem[];
		groups?: SelectGroup[];
		placeholder?: string;
		disabled?: boolean;
		onchange?: (value: Value) => void;
		class?: string;
	} = $props();

	let open = $state(false);
	let root: HTMLDivElement | undefined = $state();
	let trigger: HTMLButtonElement | undefined = $state();
	let panel: HTMLDivElement | undefined = $state();
	let panelStyle = $state('');

	function portal(node: HTMLElement) {
		(trigger?.closest('dialog') ?? document.body).appendChild(node);
		return {
			destroy() {
				node.remove();
			}
		};
	}

	const all = $derived(groups ? groups.flatMap((group) => group.items) : items);
	const selected = $derived<string[]>(Array.isArray(value) ? value : value ? [value] : []);
	const caption = $derived(
		selected
			.map((entry) => all.find((item) => item.value === entry)?.label ?? entry)
			.filter((label) => label)
			.join(', ')
	);

	function toggle() {
		if (disabled) return;
		open = !open;
		if (open && trigger) {
			const rect = trigger.getBoundingClientRect();
			const below = window.innerHeight - rect.bottom;
			panelStyle =
				below < 200 && rect.top > below
					? `left:${rect.left}px; bottom:${window.innerHeight - rect.top + 4}px; min-width:${rect.width}px; max-height:${rect.top - 8}px;`
					: `left:${rect.left}px; top:${rect.bottom + 4}px; min-width:${rect.width}px; max-height:${Math.min(below - 8, 320)}px;`;
		}
	}

	function pick(item: SelectItem) {
		if (multiple) {
			const next = selected.includes(item.value)
				? selected.filter((entry) => entry !== item.value)
				: [...selected, item.value];
			value = next as Value;
		} else {
			value = item.value as Value;
			open = false;
		}
		onchange?.(value);
	}
</script>

<svelte:window
	onpointerdown={(e) => {
		const target = e.target as Node;
		if (open && !root?.contains(target) && !panel?.contains(target)) open = false;
	}}
	onkeydown={(e) => {
		if (open && e.key === 'Escape') open = false;
	}}
	onresize={() => (open = false)}
	onscrollcapture={(e) => {
		if (open && e.target !== panel) open = false;
	}}
/>

{#snippet option(item: SelectItem)}
	<button
		type="button"
		class="flex w-full cursor-pointer items-center gap-2 rounded-sm py-1.5 pr-2 pl-2 text-left text-sm text-nowrap hover:bg-foreground/5"
		onclick={() => pick(item)}
	>
		<span class="flex size-3.5 items-center justify-center">
			{#if selected.includes(item.value)}
				<CheckIcon class="size-3.5" />
			{/if}
		</span>
		{item.label}
	</button>
{/snippet}

<div bind:this={root} class={cn('relative w-fit', className)}>
	<button
		bind:this={trigger}
		type="button"
		{disabled}
		aria-haspopup="listbox"
		aria-expanded={open}
		class="flex h-9 w-full cursor-pointer items-center justify-between gap-1.5 rounded-md border border-input bg-transparent py-2 pr-2 pl-2.5 text-sm whitespace-nowrap shadow-xs outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-input/30 dark:hover:bg-input/50"
		onclick={toggle}
	>
		<span class={cn('truncate', caption ? '' : 'text-muted-foreground')}>{caption || placeholder}</span>
		<ChevronDownIcon class="pointer-events-none size-4 shrink-0 text-muted-foreground" />
	</button>

	{#if open}
		<div
			bind:this={panel}
			use:portal
			role="listbox"
			aria-multiselectable={multiple}
			style={panelStyle}
			class="overflow-scroll-hidden fixed z-50 flex min-w-36 animate-in flex-col rounded-md bg-background p-1 text-foreground shadow-md ring-1 ring-foreground/10 duration-100 fade-in-0 zoom-in-95"
		>
			{#if groups}
				{#each groups as group (group.label)}
					<span class="px-2 py-1.5 text-xs text-muted-foreground">{group.label}</span>
					{#each group.items as item (item.value)}
						{@render option(item)}
					{/each}
				{/each}
			{:else}
				{#each items as item (item.value)}
					{@render option(item)}
				{:else}
					<span class="px-2 py-1.5 text-xs text-muted-foreground italic">no options</span>
				{/each}
			{/if}
		</div>
	{/if}
</div>

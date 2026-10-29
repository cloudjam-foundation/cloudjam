<!-- Normal badge with additional support for shimmering text -->
<script lang="ts">
	import { cn } from '$lib/utils';

	type Variant = 'primary' | 'filled' | 'jam' | 'destructive';

	const variants: Record<Variant, string> = {
		primary: 'glass text-primary/80',
		filled: 'glass text-foreground bg-background',
		jam: 'jam-glass text-jam/60',
		destructive: 'destructive-glass text-destructive/60'
	};

	let {
		text,
		variant,
		class: classNames,
		shimmer
	}: { text: string; variant: Variant; class?: string; shimmer?: boolean } = $props();
</script>

<div class={cn('w-max rounded-full px-3 py-[0.2rem] text-center text-xs', variants[variant])}>
	<span
		style="--shimmer-duration: 8s"
		class={cn(shimmer ? 'shimmer shimmer-color-primary-foreground/40' : '', classNames)}
	>
		{text}
	</span>
</div>

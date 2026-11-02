<script lang="ts">
	import { cn } from '$lib/utils';

	let { orientation = 'horizontal', class: className }: { orientation?: 'horizontal' | 'vertical'; class?: string } =
		$props();
</script>

<div
	role="separator"
	aria-orientation={orientation}
	class={cn('shrink-0 bg-border', orientation === 'horizontal' ? 'h-px w-full' : 'h-full w-px', className)}
></div>

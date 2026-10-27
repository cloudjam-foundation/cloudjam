<!-- Normal badge with additional support for shimmering text -->
<script lang="ts">
	let { text, shimmer }: { text: string; shimmer?: boolean } = $props();
</script>

<div class="glass w-max rounded-full px-3 py-[0.2rem] text-center text-xs">
	<span class="text-primary {shimmer ? 'shimmer shimmer-color-primary-foreground/40' : ''}"> {text} </span>
</div>

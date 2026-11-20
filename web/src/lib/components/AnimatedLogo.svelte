<script lang="ts">
	import { onMount } from 'svelte';
	import { cubicOut } from 'svelte/easing';

	let { class: className }: { class: string } = $props();

	const DURATION = 1000;
	const START_X = 0;
	const START_Y = 0;

	function fly(node: SVGGraphicsElement, { delay = 0 } = {}) {
		const box = node.getBBox();
		const dx = START_X - (box.x + box.width / 2);
		const dy = START_Y - (box.y + box.height / 2);

		return {
			duration: DURATION,
			delay,
			easing: cubicOut,
			css: (t: number, u: number) => `
      transform-box: fill-box;
      transform-origin: center;
      transform: translate(${u * dx}px, ${u * dy}px) scale(${0.3 + 0.7 * t});
      opacity: ${t};
    `
		};
	}

	let show = $state(false);
	onMount(() => (show = true));
</script>

{#if show}
	<svg id="logo" class={className} viewBox="0 0 2293 2293" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path
			in:fly={{ delay: 400 }}
			d="M339.5 858.108L637.487 562.14L637.494 562.108L1039.45 570.107L1039.44 570.111L1324.41 687.106L1273.03 813.535L1324.52 687.109L1624.51 901.609L1624.47 901.647L1839.76 1116.57L1996.41 1097.61L2218.42 1116.61L2254.41 1332.11L2254.41 1480.61L2246.72 1480.61L2254.41 1480.61L2241.91 1580.11L2174.41 1688.61L1436.91 1704.61L1436.92 1704.6L982.923 1525.61L983 1525.6L982.996 1525.6L982.933 1525.6L982.957 1525.58L679.508 1377.61L679.518 1377.6L411.023 1152.11L411.466 1152L411.02 1152.11L339.5 858.108ZM591.968 1109.32L579.357 1011.4L591.966 1109.32L591.968 1109.32Z"
			fill-rule="evenodd"
			clip-rule="evenodd"
			fill="#EBEBEB"
			fill-opacity="0.6"
		/>
		<path
			in:fly={{ delay: 500 }}
			d="M1861.03 794.109L1873.04 1023.61L1839.52 1116.11L1624.52 901.609L1324.52 687.109L1861.03 794.109Z"
			fill="#EBEBEB"
			fill-opacity="0.6"
		/>
		<path
			in:fly={{ delay: 600 }}
			d="M982.918 1525.62L1436.92 1704.62L754.918 1721.68L982.918 1525.62Z"
			fill="#EBEBEB"
			fill-opacity="0.6"
		/>
		<path
			in:fly={{ delay: 1000 }}
			d="M410.903 1152.11L679.418 1377.62V1377.62L742.418 1664.67L679.42 1377.62L982.918 1525.62L754.917 1721.62L754.896 1721.61L327.94 1732.65L371.478 1688.63L327.918 1732.62L410.903 1152.11Z"
			fill="#EBEBEB"
			fill-opacity="0.6"
		/>
		<path
			in:fly={{ delay: 700 }}
			d="M39.4114 1318.13L39.3916 1318.12L39.4051 1318.11L39.4052 1318.11L178.404 1152.62L410.903 1152.11L327.902 1732.59L327.906 1732.61L327.9 1732.61L327.897 1732.62L327.893 1732.6L126.404 1682.61L39.4114 1318.13Z"
			fill="#EBEBEB"
			fill-opacity="0.6"
		/>
		<path
			in:fly={{ delay: 100 }}
			d="M513.719 1544.29L754.719 1721.29L327.725 1732.33L513.719 1544.29Z"
			fill="#717171"
			fill-opacity="0.7"
		/>
		<path
			in:fly={{ delay: 300 }}
			d="M679.205 1377.31L982.702 1525.3L754.702 1721.3L679.205 1377.31Z"
			fill="#7B7B7B"
			fill-opacity="0.55"
		/>
		<path
			in:fly={{ delay: 400 }}
			d="M866.742 647.789L1212.22 962.288L591.747 1109.29L866.742 647.789Z"
			fill="#696969"
			fill-opacity="0.55"
		/>
		<path
			in:fly={{ delay: 1000 }}
			d="M1324.3 686.791L1624.29 901.292L982.79 1525.29L1324.3 686.791Z"
			fill="#7B7B7B"
			fill-opacity="0.55"
		/>
		<path
			in:fly={{ delay: 1100 }}
			d="M39.1901 1317.79L264.691 1427.79L327.691 1732.29L126.189 1682.29L39.1901 1317.79Z"
			fill="#7B7B7B"
			fill-opacity="0.55"
		/>
		<path
			in:fly={{ delay: 400 }}
			d="M410.807 1151.79L1212.22 962.288L679.309 1377.29L410.807 1151.79Z"
			fill="#7E7E7E"
			fill-opacity="0.6"
		/>
		<path
			in:fly={{ delay: 100 }}
			d="M410.687 1151.8L679.202 1377.31L327.702 1732.31L410.687 1151.8Z"
			fill="#AAAAAA"
			fill-opacity="0.6"
		/>
		<path
			in:fly={{ delay: 800 }}
			d="M679.29 1377.29L1212.22 962.289L982.79 1525.29L679.29 1377.29Z"
			fill="#AAAAAA"
			fill-opacity="0.6"
		/>
		<path
			in:fly={{ delay: 600 }}
			d="M562.297 880.291L637.277 561.79L1039.23 569.789L866.759 647.79L591.788 1109.29L562.297 880.291Z"
			fill="#AAAAAA"
			fill-opacity="0.6"
		/>
		<path
			in:fly={{ delay: 900 }}
			d="M982.702 1525.31L1436.7 1704.31L754.702 1721.36L982.702 1525.31Z"
			fill="#636363"
			fill-opacity="0.67"
		/>
		<path
			in:fly={{ delay: 700 }}
			d="M679.201 1377.3L754.702 1721.3L513.702 1544.3L679.201 1377.3Z"
			fill="#4F4F4F"
			fill-opacity="0.6"
		/>
		<path
			in:fly={{ delay: 200 }}
			d="M637.302 561.791L562.301 880.291L591.786 1109.29L410.803 1151.79L339.284 857.79L637.302 561.791Z"
			fill="#4F4F4F"
			fill-opacity="0.6"
		/>
		<path
			in:fly={{ delay: 500 }}
			d="M1624.22 901.288L1605.22 1287.29L982.719 1525.29L1624.22 901.288Z"
			fill="#5E5E5E"
			fill-opacity="0.33"
		/>
		<path
			in:fly={{ delay: 900 }}
			d="M410.687 1151.8L327.681 1732.31L264.676 1427.81L39.1757 1317.81L178.188 1152.31L410.687 1151.8Z"
			fill="#5E5E5E"
			fill-opacity="0.33"
		/>
		<path
			in:fly={{ delay: 1000 }}
			d="M2204.21 1480.29L1436.71 1704.29L982.709 1525.29L2204.21 1480.29Z"
			fill="#5E5E5E"
			fill-opacity="0.33"
		/>
		<path
			in:fly={{ delay: 1200 }}
			d="M1860.81 793.791L1872.82 1023.29L1839.31 1115.79L1624.31 901.291L1324.3 686.791L1860.81 793.791Z"
			fill="#5E5E5E"
			fill-opacity="0.33"
		/>
		<path
			in:fly={{ delay: 600 }}
			d="M1605.22 1287.29L2204.22 1480.29L982.719 1525.29L1605.22 1287.29Z"
			fill="#595959"
			fill-opacity="0.7"
		/>
		<path
			in:fly={{ delay: 400 }}
			d="M2218.21 1116.29L2254.19 1331.79L2254.19 1480.29L2204.21 1480.29L1839.21 1116.29L1996.19 1097.29L2218.21 1116.29Z"
			fill="#595959"
			fill-opacity="0.7"
		/>
		<path
			in:fly={{ delay: 800 }}
			d="M1624.22 901.288L2204.21 1480.29L1605.22 1287.29L1624.22 901.288Z"
			fill="#AAAAAA"
			fill-opacity="0.6"
		/>
		<path
			in:fly={{ delay: 700 }}
			d="M1039.22 569.79L1324.2 686.788L1212.22 962.29L866.723 647.79L1039.22 569.79Z"
			fill="#AAAAAA"
			fill-opacity="0.6"
		/>
		<path
			in:fly={{ delay: 1200 }}
			d="M2204.19 1480.29L2254.19 1480.29L2241.7 1579.79L2174.2 1688.29L1436.7 1704.29L2204.19 1480.29Z"
			fill="#AAAAAA"
			fill-opacity="0.6"
		/>
	</svg>
{/if}

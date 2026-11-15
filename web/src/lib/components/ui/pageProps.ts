import type { Snippet } from 'svelte';

export type PageProps = {
	title: string;
	headers?: Snippet[];
	back: () => void | Promise<void>;
};
import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

// createState is used to construct a svelte state outside of svelte files.
export function createState<T>(initial: T) {
	let state = $state({ value: initial });
	return state;
}

// cn merges input tailwind classes in correct order and tailwind layering.
export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

// converts the provided image to a jpeg data url.
export function toDataURL(image: Uint8Array): string {
	let raw = '';
	for (let offset = 0; offset < image.length; offset += 8192) {
		raw += String.fromCharCode(...image.subarray(offset, offset + 8192));
	}
	return `data:image/jpeg;base64,${btoa(raw)}`;
}

// converts the provided raw sha256 hash into a human readable format.
export function toDigest(hash: Uint8Array): string {
	if (!hash) return '';
	return `sha256:${Array.from(hash, (b) => b.toString(16).padStart(2, '0')).join('')}`;
}

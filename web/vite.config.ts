import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { mdsvex } from 'mdsvex';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			adapter: adapter({
				fallback: 'index.html',
				precompress: false,
				strict: true
			}),
			preprocess: [mdsvex()],
			extensions: ['.svelte', '.svx'],
			// Temporary fix
			// TODO: Migrate to #lib
			alias: { $lib: 'src/lib' }
		})
	],
	optimizeDeps: {
		include: ['@lucide/svelte', 'bits-ui', 'clsx', 'tailwind-merge']
	}
});

import { getContext, setContext } from 'svelte';
import { SvelteSet } from 'svelte/reactivity';

export type FormMethods = 'dialog' | 'get' | 'post' | 'DIALOG' | 'GET' | 'POST' | undefined | null;

const key = Symbol('SmartForm');

type Field = {
	validate: () => string | undefined;
	focus: () => void;
};

export class SmartFormClass {
	live = $state(false);

	#fields = new SvelteSet<Field>();

	register(field: Field) {
		this.#fields.add(field);

		// Allows the form to deregister
		// when the component deletes
		return () => {
			this.#fields.delete(field);
		};
	}

	validate() {
		let invalid: Field | undefined = undefined;
		for (const field of this.#fields) {
			if (field.validate() !== undefined && !invalid) invalid = field;
		}
		invalid?.focus();
		return !invalid;
	}
}

export function registerSmartForm(form: SmartFormClass) {
	return setContext(key, form);
}

export function getSmartForm(): SmartFormClass | undefined {
	return getContext(key);
}

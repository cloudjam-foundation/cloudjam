import { getContext, setContext } from 'svelte';
import { MediaQuery } from 'svelte/reactivity';

const DEFAULT_MOBILE_BREAKPOINT = 768;

export class IsMobile extends MediaQuery {
	constructor(breakpoint: number = DEFAULT_MOBILE_BREAKPOINT) {
		super(`max-width: ${breakpoint - 1}px`);
	}
}

const KEY = Symbol('sidebar');
const SHORTCUT = 'b';
const STORAGE = 'sidebar_open';

// SidebarState is shared through the context by the Sidebar shell and read by its children.
// On desktop `open` toggles between the expanded sidebar and the icon rail (persisted),
// on mobile `openMobile` toggles the drawer.
export class SidebarState {
	open = $state(true);
	openMobile = $state(false);
	#mobile = new IsMobile();

	constructor() {
		try {
			const stored = localStorage.getItem(STORAGE);
			if (stored !== null) this.open = stored === 'true';
		} catch {
			// no storage available, keep the default
		}
	}

	get isMobile() {
		return this.#mobile.current;
	}

	// expanded reports whether the full menu (icons and labels) is visible.
	get expanded() {
		return this.isMobile ? this.openMobile : this.open;
	}

	setOpen = (value: boolean) => {
		if (this.isMobile) {
			this.openMobile = value;
			return;
		}
		this.open = value;
		try {
			localStorage.setItem(STORAGE, String(value));
		} catch {
			// ignore
		}
	};

	toggle = () => this.setOpen(!(this.isMobile ? this.openMobile : this.open));

	handleShortcut = (e: KeyboardEvent) => {
		if (e.key === SHORTCUT && (e.metaKey || e.ctrlKey)) {
			e.preventDefault();
			this.toggle();
		}
	};
}

export function setSidebar(): SidebarState {
	return setContext(KEY, new SidebarState());
}

export function useSidebar(): SidebarState {
	return getContext<SidebarState>(KEY);
}

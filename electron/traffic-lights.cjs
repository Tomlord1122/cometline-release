/** Keep in sync with --content-panel-inset in src/app.css */
const CONTENT_PANEL_INSET = 6;

/** hiddenInset default-ish anchor when the sidebar column is visible */
const TRAFFIC_X_OPEN = 16;
/** Shift right when the main panel gains shell left inset (sidebar collapsed) */
const TRAFFIC_X_CLOSED = TRAFFIC_X_OPEN + CONTENT_PANEL_INSET;
const TRAFFIC_Y = 18;
const ANIMATION_MS = 240;

function easeSmooth(t) {
	return 1 - (1 - t) ** 3;
}

function canControlTrafficLights(window) {
	return (
		process.platform === 'darwin' &&
		window &&
		!window.isDestroyed() &&
		typeof window.setWindowButtonPosition === 'function'
	);
}

function trafficXForSidebar(open) {
	return open ? TRAFFIC_X_OPEN : TRAFFIC_X_CLOSED;
}

function setTrafficLightPosition(window, x) {
	window.setWindowButtonPosition({ x: Math.round(x), y: TRAFFIC_Y });
}

let activeAnimation = null;

function stopTrafficLightAnimation() {
	if (!activeAnimation) return;
	clearInterval(activeAnimation);
	activeAnimation = null;
}

function animateTrafficLights(window, open) {
	if (!canControlTrafficLights(window)) return;

	stopTrafficLightAnimation();

	const targetX = trafficXForSidebar(open);
	const current = window.getWindowButtonPosition?.();
	const startX = current?.x ?? targetX;

	if (Math.abs(startX - targetX) < 0.5) {
		setTrafficLightPosition(window, targetX);
		return;
	}

	const started = Date.now();
	activeAnimation = setInterval(() => {
		if (!canControlTrafficLights(window)) {
			stopTrafficLightAnimation();
			return;
		}

		const t = Math.min((Date.now() - started) / ANIMATION_MS, 1);
		const x = startX + (targetX - startX) * easeSmooth(t);
		setTrafficLightPosition(window, x);

		if (t >= 1) stopTrafficLightAnimation();
	}, 16);
}

function syncTrafficLights(window, open, { animate = true } = {}) {
	if (!canControlTrafficLights(window)) return;

	stopTrafficLightAnimation();

	if (animate) {
		animateTrafficLights(window, open);
		return;
	}

	setTrafficLightPosition(window, trafficXForSidebar(open));
}

module.exports = {
	syncTrafficLights,
	trafficXForSidebar,
	TRAFFIC_Y
};

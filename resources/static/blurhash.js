(() => {
    "use strict";
    const t = (t, n, e) => {
            let o = 0;
            for (; n < e;) o *= 83, o += "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz#$%*+,-.:;=?@[]^_{|}~".indexOf(t[n++]);
            return o
        }, n = Math.pow, e = Math.PI, o = 2 * e, r = 3294.6, a = 269.025,
        c = t => t > 10.31475 ? n(t / a + .052132, 2.4) : t / r,
        u = t => ~~(t > 1227e-8 ? a * n(t, .416666) - 13.025 : t * r + 1), s = t => (t < 0 ? -1 : 1) * t * t, f = t => {
            for (t += e / 2; t > e;) t -= o;
            const n = 1.27323954 * t - .405284735 * s(t);
            return .225 * (s(n) - n) + n
        };

    //             <canvas id="canvas" width="32" height="32"></canvas>

    function blurHash(hash, targetCanvas) {
        const i = targetCanvas.getContext("2d")
        const n = hash;
        if (!n) return;
        const o = function (n, o, r, a) {
            const d = t(n, 0, 1), i = d % 9 + 1, l = 1 + ~~(d / 9), m = i * l;
            let g = 0, I = 0, h = 0, p = 0, w = 0, y = 0, v = 0, E = 0, x = 0, A = 0, B = 0, C = 0, D = 0, M = 0;
            const b = (t(n, 1, 2) + 1) / 13446 * (1 | a), F = new Float64Array(3 * m), L = function (n) {
                const e = t(n, 2, 6);
                return [e >> 16, e >> 8 & 255, 255 & e]
            }(n);
            for (g = 0; g < 3; g++) F[g] = c(L[g]);
            for (g = 1; g < m; g++) M = t(n, 4 + 2 * g, 6 + 2 * g), F[3 * g] = s(~~(M / 361) - 9) * b, F[3 * g + 1] = s(~~(M / 19) % 19 - 9) * b, F[3 * g + 2] = s(M % 19 - 9) * b;
            const O = 4 * o, P = new Uint8ClampedArray(O * r);
            for (p = 0; p < r; p++) for (C = e * p / r, h = 0; h < o; h++) {
                for (w = 0, y = 0, v = 0, D = e * h / o, I = 0; I < l; I++) for (x = f(C * I), g = 0; g < i; g++) E = f(D * g) * x, A = 3 * (g + I * i), w += F[A] * E, y += F[A + 1] * E, v += F[A + 2] * E;
                B = 4 * h + p * O, P[B] = u(w), P[B + 1] = u(y), P[B + 2] = u(v), P[B + 3] = 255
            }
            return P
        }(n, 32, 32), r = new ImageData(o, 32, 32);
        i.putImageData(r, 0, 0)
    }

    window.blurHash = blurHash;
})();
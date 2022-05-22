import typescript from 'rollup-plugin-typescript2';
import resolve from 'rollup-plugin-node-resolve';
import commonjs from 'rollup-plugin-commonjs';


export default {
    input: './src/index.ts',
    output: [{
        file: './dist/index.js',
        format: 'cjs',
        sourcemap: true,
    }, {
        file: './dist/index.mjs',
        format: 'esm',
        sourcemap: true,
    }],
    plugins: [
        typescript(),
        resolve({browser: true}),
        commonjs(),
    ],
}

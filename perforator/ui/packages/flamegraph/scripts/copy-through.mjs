import fs from 'node:fs'
import glob from 'fast-glob'

// TODO switch to fs.globSync when switch to a fresher node.js
glob.globSync('lib/components/**/*.css').forEach(filepath => {
    const targetPath = filepath.replace('lib', 'dist')
    console.log(`copying ${filepath} to ${targetPath}`)
    // implicitly relies on the fact that tsc will create the folder structure
    // otherwise will break if no folder structure like dist/components/componentName present
    fs.copyFileSync(filepath, targetPath)
})

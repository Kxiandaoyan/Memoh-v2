import fs from 'fs'
import path from 'path'

const rootDir = path.resolve(import.meta.dirname, '../src')
const readDir = path.resolve(rootDir, './components')
const outputDir = path.resolve(rootDir, './index.ts')

async function readDirName(){  
  const pathList:Awaited<string[]> = await new Promise((resolve, reject) => {
    fs.readdir(readDir, (err, data) => {
      if (err) {    
         reject(err)
      }
      resolve(data)
     })
   })
   return pathList 
}

async function writeExportFile(pathList: string[]) {
  const pathListStr = pathList.map(fileName => {
    return `export * from './components/${fileName}/index'`
  })
  await new Promise((resolve, reject) => {
    fs.writeFile(outputDir, pathListStr.join('\r\n'), (err) => {
      if (err) {
        reject(err)
      }
      resolve(undefined)
    })   
  })
}

async function generate() {
  try {
    const list = await readDirName()
    writeExportFile(list)
  } catch(error) {
    console.error(error)
  }
}

generate()
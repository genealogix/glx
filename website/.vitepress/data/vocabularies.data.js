import { readFileSync } from 'fs'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)

export default {
  watch: ['../../../specification/5-standard-vocabularies/*.glx'],
  load() {
    const vocabDir = resolve(__dirname, '../../../specification/5-standard-vocabularies')

    const vocabularies = {
      'citation-properties': readFileSync(resolve(vocabDir, 'citation-properties.glx'), 'utf-8'),
      'confidence-levels': readFileSync(resolve(vocabDir, 'confidence-levels.glx'), 'utf-8'),
      'event-properties': readFileSync(resolve(vocabDir, 'event-properties.glx'), 'utf-8'),
      'event-types': readFileSync(resolve(vocabDir, 'event-types.glx'), 'utf-8'),
      'gender-types': readFileSync(resolve(vocabDir, 'gender-types.glx'), 'utf-8'),
      'media-properties': readFileSync(resolve(vocabDir, 'media-properties.glx'), 'utf-8'),
      'media-types': readFileSync(resolve(vocabDir, 'media-types.glx'), 'utf-8'),
      'participant-roles': readFileSync(resolve(vocabDir, 'participant-roles.glx'), 'utf-8'),
      'person-properties': readFileSync(resolve(vocabDir, 'person-properties.glx'), 'utf-8'),
      'place-properties': readFileSync(resolve(vocabDir, 'place-properties.glx'), 'utf-8'),
      'place-types': readFileSync(resolve(vocabDir, 'place-types.glx'), 'utf-8'),
      'relationship-properties': readFileSync(
        resolve(vocabDir, 'relationship-properties.glx'),
        'utf-8'
      ),
      'relationship-types': readFileSync(resolve(vocabDir, 'relationship-types.glx'), 'utf-8'),
      'repository-properties': readFileSync(
        resolve(vocabDir, 'repository-properties.glx'),
        'utf-8'
      ),
      'repository-types': readFileSync(resolve(vocabDir, 'repository-types.glx'), 'utf-8'),
      'source-properties': readFileSync(resolve(vocabDir, 'source-properties.glx'), 'utf-8'),
      'source-types': readFileSync(resolve(vocabDir, 'source-types.glx'), 'utf-8')
    }

    return vocabularies
  }
}

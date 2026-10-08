import { i18n } from 'src/utils/i18n';

import en from './en.json';


const keyset = 'components/TaskCard/EditableTaskQuery';

i18n.registerKeyset('en', keyset, en);

export default i18n.keyset<keyof typeof en>(keyset);

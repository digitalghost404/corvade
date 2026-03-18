import { CorvadeWS } from './ws';

let _instance: CorvadeWS | null = null;

export function getWS(): CorvadeWS {
  if (!_instance) {
    _instance = new CorvadeWS();
  }
  return _instance;
}

#!/usr/bin/env node

import fs from 'fs';
import { v4 as uuidv4 } from 'uuid';
import Folio from '@indexdata/foliojs';

const op = process.argv[2];
if (process.argv.length !== 3 || (op !== 'get' && op !== 'set' && op !== 'write-settings')) {
  console.error('Usage:', process.argv[1], '<get|set|write-settings>');
  process.exit(1);
}

const [_, session] = await Folio.defaultSetup();
if (op === 'get') {
  const body = await session.folioFetch('/ldp/config');
  console.log(JSON.stringify(body, null, 2));
  session.close();
  process.exit(0);
}

// One of the two setting operations
var data = fs.readFileSync(process.stdin.fd, 'utf-8');
const json = JSON.parse(data)
for (let i = 0; i < json.length; i++) {
  const rec = { ...json[i] };
  if (typeof rec.value !== 'string') {
    rec.value = JSON.stringify(rec.value);
  }

  if (op === 'set') {
    await session.folioFetch(`/ldp/config/${rec.key}`, {
      method: 'PUT',
      json: rec,
    });
  } else {
    const id = uuidv4();
    await session.folioFetch(`/settings/entries`, {
      method: 'POST',
      json: {
        id,
        scope: 'ui-ldp.admin',
        key: rec.key,
        value: rec.value,
      }
    });
  }
  console.log('record', i, '--', rec);
}

session.close();

import { expect, test } from '@jest/globals';

import { bracketColorizer, colorize } from './bracketColorizer.tsx';


test('bracketColorizer', () => {
    expect(bracketColorizer('std::function<void ()>::operator()() const')).toMatchInlineSnapshot(`
[
  {
    "depth": 1,
    "firstChar": "(",
    "firstIndex": 19,
    "lastChar": ")",
    "lastIndex": 20,
  },
  {
    "depth": 0,
    "firstChar": "<",
    "firstIndex": 13,
    "lastChar": ">",
    "lastIndex": 21,
  },
  {
    "depth": 0,
    "firstChar": "(",
    "firstIndex": 32,
    "lastChar": ")",
    "lastIndex": 33,
  },
  {
    "depth": 0,
    "firstChar": "(",
    "firstIndex": 34,
    "lastChar": ")",
    "lastIndex": 35,
  },
]
`);
});

test('colorize', () => {
    expect(colorize('std::function<void ()>::operator()() const')).toMatchInlineSnapshot(`
[
  "std::function",
  <span
    className="color-0"
  >
    &lt;
  </span>,
  "void ",
  <span
    className="color-1"
  >
    (
  </span>,
  <span
    className="color-1"
  >
    )
  </span>,
  <span
    className="color-0"
  >
    &gt;
  </span>,
  "::operator",
  <span
    className="color-0"
  >
    (
  </span>,
  <span
    className="color-0"
  >
    )
  </span>,
  <span
    className="color-0"
  >
    (
  </span>,
  <span
    className="color-0"
  >
    )
  </span>,
  " const",
]
`);
});

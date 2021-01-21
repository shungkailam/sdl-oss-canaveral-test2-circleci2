import { dSourceDstreamMatch } from './dSourceDstreamMatchUtil';
describe('dataSourceDataStreamMatch', () => {
  it('should find whether dataSource dataStream Match', () => {
    let dSource = {},
      dStream = {};
    expect(dSourceDstreamMatch(dSource, dStream)).toBeFalsy;

    dSource = {
      selectors: [
        {
          'id:': '27d5de74-7a68-4ec6-b573-1f70734f7952',
          value: 'LAX',
        },
        {
          id: '0bf4230a-49a9-4327-8d36-715a1bc14a84',
          value: 'Terminal',
        },
      ],
    };
    dStream = {};
    expect(dSourceDstreamMatch(dSource, dStream)).toBeFalsy;

    dSource = {};
    dStream = {
      originSelectors: [
        {
          'id:': '27d5de74-7a68-4ec6-b573-1f70734f7952',
          value: 'LAX',
        },
        {
          id: '0bf4230a-49a9-4327-8d36-715a1bc14a84',
          value: 'Terminal',
        },
      ],
    };
    expect(dSourceDstreamMatch(dSource, dStream)).toBeFalsy;

    dSource = {
      selectors: [
        {
          'id:': '27d5de74-7a68-4ec6-b573-1f70734f7952',
          value: 'LAX',
        },
        {
          id: '0bf4230a-49a9-4327-8d36-715a1bc14a84',
          value: 'Parking',
        },
        {
          id: '0bf4230a-49a9-4327-8d36-715a1bc14a84',
          value: 'Terminal',
        },
      ],
    };

    dStream = {
      originSelectors: [
        {
          'id:': '27d5de74-7a68-4ec6-b573-1f70734f7952',
          value: 'LAX',
        },
        {
          id: '0bf4230a-49a9-4327-8d36-715a1bc14a84',
          value: 'Terminal',
        },
      ],
    };
    expect(dSourceDstreamMatch(dSource, dStream)).toBeTruthy;

    dSource = {
      selectors: [
        {
          'id:': '27d5de74-7a68-4ec6-b573-1f70734f7952',
          value: 'LAX',
        },
        {
          id: '0bf4230a-49a9-4327-8d36-715a1bc14a87',
          value: 'Parking',
        },
        {
          id: '0bf4230a-49a9-4327-8d36-715a1bc14a84',
          value: 'Terminal',
        },
      ],
    };

    dStream = {
      originSelectors: [
        {
          'id:': '27d5de74-7a68-4ec6-b573-1f70734f7952',
          value: 'LAX',
        },
        {
          id: '0bf4230a-49a9-4327-8d36-715a1bc14a84',
          value: 'Terminal',
        },
      ],
    };
    expect(dSourceDstreamMatch(dSource, dStream)).toBeFalsy;
  });
});

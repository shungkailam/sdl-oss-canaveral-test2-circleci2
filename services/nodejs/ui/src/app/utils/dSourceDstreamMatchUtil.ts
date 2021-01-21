// find Whether a dataStream is included in a data source
export function dSourceDstreamMatch(dSource, dStream) {
  if (!dSource || !dStream) {
    return false;
  }

  let dSourceSel = dSource['selectors'],
    dStreamSel = dStream['originSelectors'],
    matchedCategoryIds = {},
    streamCategoryIds = {};

  if (!dSourceSel || !dStreamSel) {
    return false;
  }

  dStreamSel.forEach(dst => {
    let dStreamSelId = dst.id,
      dStreamSelVal = dst.value;
    if (!streamCategoryIds[dStreamSelId]) {
      streamCategoryIds[dStreamSelId] = true;
    }
    if (!matchedCategoryIds[dStreamSelId]) {
      for (let i = 0; i < dSourceSel.length; i++) {
        let catId = dSourceSel[i].id,
          catVal = dSourceSel[i].value;
        if (catId === dStreamSelId && catVal === dStreamSelVal) {
          matchedCategoryIds[dStreamSelId] = true;
          break;
        }
      }
    }
  });
  if (
    Object.keys(matchedCategoryIds).length ===
    Object.keys(streamCategoryIds).length
  ) {
    return true;
  }
  return false;
}

export const DIFFICULTY_EASY_THRESHOLD = 0.7;
export const DIFFICULTY_HARD_THRESHOLD = 0.3;
export const DISCRIMINATION_POOR_THRESHOLD = 0.2;
export const MIN_SAMPLE_SIZE = 20;

export function getDifficultyLevel(difficulty, hasEnoughData) {
  if (!hasEnoughData || difficulty == null) {
    return 'no_data';
  }
  if (difficulty >= DIFFICULTY_EASY_THRESHOLD) {
    return 'easy';
  }
  if (difficulty <= DIFFICULTY_HARD_THRESHOLD) {
    return 'hard';
  }
  return 'medium';
}

export function getDifficultyLabel(level) {
  switch (level) {
    case 'easy':
      return '易';
    case 'medium':
      return '中';
    case 'hard':
      return '难';
    case 'no_data':
    default:
      return '数据不足';
  }
}

export function getDifficultyBadgeClass(level) {
  switch (level) {
    case 'easy':
      return 'badge-success';
    case 'medium':
      return 'badge-warning';
    case 'hard':
      return 'badge-error';
    case 'no_data':
    default:
      return 'badge-ghost';
  }
}

export function formatDifficultyValue(difficulty, hasEnoughData) {
  if (!hasEnoughData || difficulty == null) {
    return '-';
  }
  return (difficulty * 100).toFixed(1) + '%';
}

export function getDiscriminationLevel(discrimination, hasEnoughData) {
  if (!hasEnoughData || discrimination == null) {
    return 'no_data';
  }
  if (discrimination >= 0.4) {
    return 'good';
  }
  if (discrimination >= 0.2) {
    return 'fair';
  }
  return 'poor';
}

export function getDiscriminationLabel(level) {
  switch (level) {
    case 'good':
      return '良好';
    case 'fair':
      return '一般';
    case 'poor':
      return '较差';
    case 'no_data':
    default:
      return '-';
  }
}

export function isAbnormalQuestion(stats) {
  if (!stats || !stats.hasEnoughData) {
    return false;
  }
  const difficulty = stats.difficulty;
  const discrimination = stats.discrimination;
  if (difficulty == null || discrimination == null) {
    return false;
  }
  return (
    difficulty <= DIFFICULTY_HARD_THRESHOLD ||
    difficulty >= DIFFICULTY_EASY_THRESHOLD ||
    discrimination <= DISCRIMINATION_POOR_THRESHOLD
  );
}

export function getAbnormalTypes(stats) {
  if (!stats || !stats.hasEnoughData) {
    return [];
  }
  const types = [];
  if (stats.difficulty != null && stats.difficulty <= DIFFICULTY_HARD_THRESHOLD) {
    types.push('过难');
  }
  if (stats.difficulty != null && stats.difficulty >= DIFFICULTY_EASY_THRESHOLD) {
    types.push('过易');
  }
  if (stats.discrimination != null && stats.discrimination <= DISCRIMINATION_POOR_THRESHOLD) {
    types.push('区分度差');
  }
  return types;
}

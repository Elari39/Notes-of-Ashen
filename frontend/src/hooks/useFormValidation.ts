import { useCallback, useMemo, useRef, useState } from 'react';
import { usePreferenceStore } from '../store/preferences';
import { formatText, translate, type TranslationKey } from '../i18n';

export type ValidationRule =
  | { type: 'required' }
  | { type: 'minLength'; value: number }
  | { type: 'maxLength'; value: number }
  | { type: 'pattern'; value: RegExp; key?: TranslationKey }
  | { type: 'match'; field: string; key?: TranslationKey };

export type FieldRules<T> = {
  [K in keyof T]?: ValidationRule[];
};

export type ValidationErrors<T> = Partial<Record<keyof T, string>>;

const ruleMessage = (
  language: 'zh' | 'en',
  rule: ValidationRule,
): string => {
  switch (rule.type) {
    case 'required':
      return translate(language, 'validation.required');
    case 'minLength':
      return formatText(translate(language, 'validation.minLength'), { n: rule.value });
    case 'maxLength':
      return formatText(translate(language, 'validation.maxLength'), { n: rule.value });
    case 'pattern':
      return rule.key ? translate(language, rule.key) : translate(language, 'validation.required');
    case 'match':
      return rule.key ? translate(language, rule.key) : translate(language, 'validation.passwordMismatch');
    default:
      return '';
  }
};

const validateField = <T extends Record<string, string>>(
  language: 'zh' | 'en',
  field: keyof T,
  values: T,
  rules: FieldRules<T>,
): string => {
  const fieldRules = rules[field];
  if (!fieldRules) return '';
  const value = values[field] ?? '';
  for (const rule of fieldRules) {
    switch (rule.type) {
      case 'required':
        if (!value.trim()) return ruleMessage(language, rule);
        break;
      case 'minLength':
        if (value.length > 0 && value.length < rule.value) return ruleMessage(language, rule);
        break;
      case 'maxLength':
        if (value.length > rule.value) return ruleMessage(language, rule);
        break;
      case 'pattern':
        if (value.length > 0 && !rule.value.test(value)) return ruleMessage(language, rule);
        break;
      case 'match': {
        const matched = values[rule.field as keyof T] ?? '';
        if (value !== matched) return ruleMessage(language, rule);
        break;
      }
      default:
        break;
    }
  }
  return '';
};

/**
 * 声明式表单校验 hook。切换语言时已校验字段的错误文案会即时刷新。
 */
export function useFormValidation<T extends Record<string, string>>(values: T, rules: FieldRules<T>) {
  const language = usePreferenceStore((state) => state.language);
  const [touched, setTouched] = useState<Partial<Record<keyof T, boolean>>>({});

  // 调用方常传入内联对象字面量作为 values，每次 render 都是新引用，会让 useMemo 失效。
  // 这里对 values 做字段级浅比较：字段未变则复用上一次的引用，使 memo 真正生效。
  const valuesRef = useRef<T>(values);
  if (!shallowEqualByKeys(valuesRef.current, values, rules)) {
    valuesRef.current = values;
  }
  const stableValues = valuesRef.current;

  // 依赖 language，切换语言后重算所有已 touched 字段的错误文案
  const errors = useMemo<ValidationErrors<T>>(() => {
    const result: ValidationErrors<T> = {};
    (Object.keys(rules) as (keyof T)[]).forEach((field) => {
      if (touched[field]) {
        const message = validateField(language, field, stableValues, rules);
        if (message) result[field] = message;
      }
    });
    return result;
  }, [stableValues, touched, language, rules]);

  const setFieldTouched = useCallback((field: keyof T, isTouched = true) => {
    setTouched((prev) => ({ ...prev, [field]: isTouched }));
  }, []);

  const validateFieldOnly = useCallback(
    (field: keyof T): boolean => {
      const message = validateField(language, field, stableValues, rules);
      setTouched((prev) => ({ ...prev, [field]: true }));
      return !message;
    },
    [language, stableValues, rules],
  );

  const validate = useCallback((): boolean => {
    let allValid = true;
    const nextTouched: Partial<Record<keyof T, boolean>> = {};
    (Object.keys(rules) as (keyof T)[]).forEach((field) => {
      nextTouched[field] = true;
      const message = validateField(language, field, stableValues, rules);
      if (message) allValid = false;
    });
    setTouched(nextTouched);
    return allValid;
  }, [language, stableValues, rules]);

  const resetTouched = useCallback(() => setTouched({}), []);

  return {
    errors,
    touched,
    validateField: validateFieldOnly,
    validate,
    setFieldTouched,
    resetTouched,
  };
}

// 按 rules 涉及的字段做浅比较（rules 的 key 即需校验的字段集合）。
const shallowEqualByKeys = <T extends Record<string, string>>(
  prev: T,
  next: T,
  rules: FieldRules<T>,
): boolean => {
  const keys = Object.keys(rules) as (keyof T)[];
  for (const key of keys) {
    if ((prev[key] ?? '') !== (next[key] ?? '')) {
      return false;
    }
  }
  return true;
};

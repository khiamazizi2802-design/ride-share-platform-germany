import React from 'react';

type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger';
type ButtonSize = 'sm' | 'md' | 'lg';

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  loading?: boolean;
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
  children: React.ReactNode;
}

const variantStyles: Record<ButtonVariant, string> = {
  primary:
    'bg-[#22C55E] text-white border border-transparent hover:bg-[#16A34A] active:bg-[#15803D] focus-visible:ring-[#22C55E] disabled:bg-[#86EFAC] disabled:cursor-not-allowed',
  secondary:
    'bg-[#0F172A] text-white border border-transparent hover:bg-[#1E293B] active:bg-[#334155] focus-visible:ring-[#0F172A] disabled:bg-[#94A3B8] disabled:cursor-not-allowed',
  ghost:
    'bg-transparent text-[#0F172A] border border-[#CBD5E1] hover:bg-[#F1F5F9] active:bg-[#E2E8F0] focus-visible:ring-[#94A3B8] disabled:text-[#94A3B8] disabled:border-[#E2E8F0] disabled:cursor-not-allowed',
  danger:
    'bg-[#DC2626] text-white border border-transparent hover:bg-[#B91C1C] active:bg-[#991B1B] focus-visible:ring-[#DC2626] disabled:bg-[#FCA5A5] disabled:cursor-not-allowed',
};

const sizeStyles: Record<ButtonSize, string> = {
  sm: 'h-8 px-3 text-[0.875rem] gap-1.5 rounded-md',
  md: 'h-10 px-4 text-[1rem] gap-2 rounded-lg',
  lg: 'h-12 px-6 text-[1.125rem] gap-2.5 rounded-xl',
};

const spinnerSizeStyles: Record<ButtonSize, string> = {
  sm: 'w-3.5 h-3.5',
  md: 'w-4 h-4',
  lg: 'w-5 h-5',
};

const Spinner: React.FC<{ sizeClass: string }> = ({ sizeClass }) => (
  <svg
    className={`animate-spin ${sizeClass}`}
    xmlns="http://www.w3.org/2000/svg"
    fill="none"
    viewBox="0 0 24 24"
    aria-hidden="true"
  >
    <circle
      className="opacity-25"
      cx="12"
      cy="12"
      r="10"
      stroke="currentColor"
      strokeWidth="4"
    />
    <path
      className="opacity-75"
      fill="currentColor"
      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
    />
  </svg>
);

export const Button: React.FC<ButtonProps> = ({
  variant = 'primary',
  size = 'md',
  loading = false,
  leftIcon,
  rightIcon,
  children,
  disabled,
  className = '',
  ...rest
}) => {
  const isDisabled = disabled || loading;

  return (
    <button
      disabled={isDisabled}
      aria-busy={loading}
      className={[
        'inline-flex items-center justify-center font-medium transition-all duration-150',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2',
        'select-none whitespace-nowrap',
        'font-[Inter,system-ui,sans-serif]',
        variantStyles[variant],
        sizeStyles[size],
        className,
      ]
        .filter(Boolean)
        .join(' ')}
      {...rest}
    >
      {loading ? (
        <>
          <Spinner sizeClass={spinnerSizeStyles[size]} />
          <span className="opacity-80">{children}</span>
        </>
      ) : (
        <>
          {leftIcon && <span className="inline-flex items-center shrink-0">{leftIcon}</span>}
          <span>{children}</span>
          {rightIcon && <span className="inline-flex items-center shrink-0">{rightIcon}</span>}
        </>
      )}
    </button>
  );
};

export default Button;

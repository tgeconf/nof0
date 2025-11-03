"use client";

/**
 * HelpButton - 悬浮帮助按钮组件
 * 固定在页面右下角，点击打开帮助文档
 */
export function HelpButton() {
  const handleClick = () => {
    window.open(
      "https://wquguru.gitbook.io/nof0",
      "_blank",
      "noopener,noreferrer",
    );
  };

  return (
    <button
      onClick={handleClick}
      aria-label="打开帮助文档"
      title="帮助文档"
      className="fixed bottom-6 right-6 z-40 flex items-center justify-center w-12 h-12 rounded-full border chip-btn shadow-lg transition-all duration-200 hover:scale-110"
      style={{
        background: "var(--panel-bg)",
        borderColor: "var(--chip-border)",
        color: "var(--btn-inactive-fg)",
        backdropFilter: "blur(8px)",
      }}
    >
      {/* 书籍图标 */}
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        width="20"
        height="20"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" />
        <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z" />
      </svg>
    </button>
  );
}

export default HelpButton;

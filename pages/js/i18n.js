document.addEventListener("DOMContentLoaded", () => {
    const langToggle = document.getElementById("lang-toggle");
    let currentLang = localStorage.getItem("pages-locale") || "en";

    const updateTexts = () => {
        document.querySelectorAll("[data-i18n]").forEach(el => {
            const key = el.getAttribute("data-i18n");
            if (translations[currentLang][key]) {
                el.innerHTML = translations[currentLang][key];
            }
        });
        document.documentElement.lang = currentLang;
    };

    if (langToggle) {
        langToggle.addEventListener("click", (e) => {
            e.preventDefault();
            currentLang = currentLang === "en" ? "cn" : "en";
            localStorage.setItem("pages-locale", currentLang);
            updateTexts();
        });
    }

    // Initial load
    updateTexts();
});
